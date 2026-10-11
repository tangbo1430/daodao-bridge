package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/account"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/adapter"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/business"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/config"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/engine"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/httpapi"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/leader"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/media"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/proxy"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/session"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
)

func main() {
	processCtx, processStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer processStop()
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	var persistence store.Store = store.NewMemoryStore()
	var queue interface {
		Enqueue(context.Context, string) error
	}
	var redisQueue *store.RedisQueue
	if cfg.StoreMode == "mysql" {
		db, err := sql.Open("mysql", cfg.MySQLDSN)
		if err != nil {
			log.Fatal(err)
		}
		if err := store.ValidateMySQL(db); err != nil {
			log.Fatal(err)
		}
		if cfg.AutoMigrate {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			if err := store.ApplySchema(ctx, db); err != nil {
				cancel()
				log.Fatal(err)
			}
			cancel()
		}
		persistence = store.NewMySQLStore(db)
	}
	if cfg.RedisURL != "" {
		var err error
		redisQueue, err = store.NewRedisQueue(cfg.RedisURL, "bridge:outbox")
		if err != nil {
			log.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := redisQueue.Ping(ctx); err != nil {
			cancel()
			log.Printf("redis unavailable; MySQL polling fallback remains active: %v", err)
		} else {
			queue = redisQueue
			cancel()
		}
	}
	tencent := adapter.TencentClient{SDKAppID: cfg.TencentSDKAppID, SecretKey: cfg.TencentSecretKey, AdminAccount: cfg.TencentAdminAccount, BaseURL: cfg.TencentRESTBaseURL, SigTTL: cfg.TencentSigTTL}
	_ = json.Unmarshal([]byte(cfg.TencentPushAndroidJSON), &tencent.PushAndroidInfo)
	var sessions *proxy.WuKongManager
	if cfg.TangWebSocketURL != "" {
		sessions = proxy.NewWuKongManager(cfg.TangWebSocketURL)
	}
	tang := adapter.TangSengClient{BaseURL: cfg.TangBaseURL, Token: cfg.TangToken, UserEndpoint: cfg.TangUserEndpoint, LoginEndpoint: cfg.TangLoginEndpoint, GroupEndpoint: cfg.TangGroupEndpoint, SendEndpoint: cfg.TangSendEndpoint, ShadowPassword: cfg.TangShadowPassword, LegacyShadowPassword: cfg.TangLegacyPassword, Manufacturer: cfg.TangManufacturer, InviteCode: cfg.TangInviteCode, ShadowOwnerUID: cfg.TangShadowOwnerUID, Username: cfg.TangUsername, Password: cfg.TangPassword, DeviceID: cfg.TangDeviceID, UserTokens: &sync.Map{}, Sessions: sessions}
	eng := &engine.Engine{Store: persistence, Tencent: &tencent, Tang: &tang, Queue: queue, MaxAttempts: cfg.MaxAttempts, TencentNoticeSender: cfg.TencentAdminAccount}
	accounts := account.New(&tang)
	accounts.SetStore(persistence)
	accounts.ConfigureRegistration(cfg.EncryptionKey)
	if endpoint := os.Getenv("BRIDGE_PHP_ACCOUNT_STATUS_URL"); endpoint != "" {
		accounts.NotifyStatus = account.PHPStatus(endpoint, cfg.InternalSharedSecret)
	}
	if endpoint := os.Getenv("BRIDGE_PHP_REGISTRATION_LOOKUP_URL"); endpoint != "" {
		accounts.RegistrationLookup = account.PHPLookup(endpoint, cfg.InternalSharedSecret)
	}
	tang.Credential = accounts.RegisteredCredential
	tang.RefreshUser = accounts.Refresh
	tang.RefreshUserIfToken = accounts.RefreshIfToken
	if sessions != nil {
		accounts.VerifyConnection = sessions.VerifyConnection
		sessions.RefreshUser = accounts.RefreshIfToken
		sessions.ConnectionFailure = accounts.ConnectionFailure
	}
	businessService := business.New(persistence, accounts, &tang, &tencent, cfg.EncryptionKey)
	businessService.Accept = eng.Accept
	eng.RecordDelivery = businessService.RecordDelivery
	eng.BusinessHandler = businessService.Handle
	eng.GroupResolver = businessService.ResolveGroupMessage
	eng.TencentScope = businessService.TencentConversation
	mediaService := &media.Service{Store: persistence, TangBase: cfg.TangBaseURL, PublicBase: os.Getenv("BRIDGE_MEDIA_PUBLIC_URL"), Secret: cfg.InternalSharedSecret}
	mediaService.Token = func(uid string) (string, bool) {
		v, ok := tang.UserTokens.Load(uid)
		token, _ := v.(string)
		return token, ok
	}
	eng.PrepareMessage = businessService.Prepare
	businessService.Media = mediaService
	if sessions != nil {
		accounts.SetProxyLifecycle(func(ctx context.Context, identity domain.Identity) error {
			log.Printf("tangseng proxy connecting tencent_uid=%s tsdd_uid=%s device_id=hwbridge-%s", identity.SourceID, identity.TargetID, identity.TargetID)
			sessions.SetDeviceID(identity.TargetID, "hwbridge-"+identity.TargetID)
			err := sessions.StartReceiveLoop(processCtx, identity.TargetID, func(eventCtx context.Context, event session.Event) error {
				log.Printf("tangseng recv uid=%s message_id=%s seq=%d channel=%s type=%d from=%s", identity.TargetID, event.MessageID, event.MessageSeq, event.ChannelID, event.ChannelType, event.FromUID)
				envelope := proxy.EventEnvelope(identity.TargetID, event)
				accepted, acceptErr := eng.Accept(eventCtx, envelope)
				if !accepted {
					return acceptErr
				}
				if acceptErr != nil {
					log.Printf("tangseng event accepted with processing error: %v", acceptErr)
				} else {
					log.Printf("tangseng event accepted message_id=%s", event.MessageID)
				}
				return nil
			})
			if err != nil {
				log.Printf("tangseng proxy disconnected tencent_uid=%s tsdd_uid=%s error=%v", identity.SourceID, identity.TargetID, err)
			}
			return err
		}, func(_ context.Context, identity domain.Identity) error {
			sessions.StopReceiveLoop(identity.TargetID)
			return nil
		})
	}
	if err := accounts.Restore(context.Background()); err != nil {
		log.Printf("account restore failed: %v", err)
	} else {
		log.Printf("account restore complete")
	}
	api := &httpapi.Server{Engine: eng, Store: persistence, SharedSecret: cfg.InternalSharedSecret, TencentSDKAppID: cfg.TencentSDKAppID, Tencent: &tencent, Tang: &tang, BridgePeerID: cfg.BridgePeerID, Accounts: accounts}
	api.Business = businessService
	api.Media = mediaService
	api.MediaSource = http.HandlerFunc(mediaService.ServeSource)
	if sessions != nil {
		api.TestSendText = func(sendCtx context.Context, fromUID, toUID, text string) (string, error) {
			log.Printf("admin test send tangseng from=%s to=%s", fromUID, toUID)
			return sessions.SendText(sendCtx, fromUID, toUID, 1, text, "admin_test_"+domain.StableID(fromUID, toUID, text, time.Now().String()))
		}
	}
	server := &http.Server{Addr: cfg.ListenAddr, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go businessService.RunSync(ctx)
	var leaderOwned atomic.Bool
	var lease *leader.Lease
	if cfg.LeaderEnabled && redisQueue != nil {
		instance := cfg.LeaderInstance
		if instance == "" {
			instance, _ = os.Hostname()
		}
		lease = &leader.Lease{Client: redisQueue.Client, Key: cfg.LeaderKey, Value: instance, TTL: time.Duration(cfg.LeaderTTLSeconds) * time.Second}
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				acquired, err := lease.Acquire(ctx)
				if err == nil && acquired {
					leaderOwned.Store(true)
				}
				if leaderOwned.Load() {
					renewed, err := lease.Renew(ctx)
					if err != nil || !renewed {
						leaderOwned.Store(false)
					}
				}
				select {
				case <-ctx.Done():
					_ = lease.Release(context.Background())
					return
				case <-ticker.C:
				}
			}
		}()
	} else {
		leaderOwned.Store(true)
	}
	if sessions != nil && cfg.TangUsername != "" && cfg.TangPassword != "" {
		go runTangReceive(ctx, &tang, sessions, eng)
	}
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.WorkerIntervalMS) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if leaderOwned.Load() {
					if _, err := eng.ProcessDue(ctx, 100); err != nil {
						log.Printf("outbox worker error: %v", err)
					}
				}
			}
		}
	}()
	go func() {
		log.Printf("bridge listening on %s", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("bridge server stopped: %v", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
	if sessions != nil {
		_ = sessions.Close()
	}
}

func runTangReceive(ctx context.Context, tang *adapter.TangSengClient, sessions *proxy.WuKongManager, eng *engine.Engine) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		uid, err := tang.Login(ctx)
		if err != nil {
			log.Printf("tangseng login failed: %v; retry in %s", err, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			if backoff < 60*time.Second {
				backoff *= 2
				if backoff > 60*time.Second {
					backoff = 60 * time.Second
				}
			}
			continue
		}
		log.Printf("tangseng PC(2) connecting uid=%s", uid)
		backoff = time.Second
		err = sessions.RunReceiveLoop(ctx, uid, func(eventCtx context.Context, event session.Event) error {
			log.Printf("tangseng recv uid=%s message_id=%s seq=%d channel=%s type=%d from=%s", uid, event.MessageID, event.MessageSeq, event.ChannelID, event.ChannelType, event.FromUID)
			envelope := proxy.EventEnvelope(uid, event)
			accepted, acceptErr := eng.Accept(eventCtx, envelope)
			if !accepted {
				log.Printf("tangseng recv rejected message_id=%s error=%v", event.MessageID, acceptErr)
				return acceptErr
			}
			// Parse failures are durably represented as an outbox/dead-letter
			// record; acknowledge them so one malformed frame cannot block the
			// account forever.
			if acceptErr != nil {
				log.Printf("tangseng event %s accepted with processing error: %v", event.MessageID, acceptErr)
			} else {
				log.Printf("tangseng event accepted message_id=%s", event.MessageID)
			}
			return nil
		})
		if err == nil || ctx.Err() != nil {
			return
		}
		log.Printf("tangseng PC(2) disconnected uid=%s: %v; reconnect in %s", uid, err, backoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		if backoff < 60*time.Second {
			backoff *= 2
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
		}
	}
}
