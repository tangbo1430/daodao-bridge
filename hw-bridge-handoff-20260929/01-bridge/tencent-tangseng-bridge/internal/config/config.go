package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TencentPushAndroidJSON string
	ListenAddr             string
	InternalSharedSecret   string
	StoreMode              string
	MySQLDSN               string
	AutoMigrate            bool
	RedisURL               string
	MaxAttempts            int
	WorkerIntervalMS       int
	TencentSDKAppID        int
	TencentSecretKey       string
	TencentAdminAccount    string
	TencentRESTBaseURL     string
	TencentSigTTL          int
	TangBaseURL            string
	TangToken              string
	TangUserEndpoint       string
	TangLoginEndpoint      string
	TangGroupEndpoint      string
	TangSendEndpoint       string
	TangFetchEndpoint      string
	TangShadowPassword     string
	TangLegacyPassword     string
	TangManufacturer       string
	TangInviteCode         string
	TangShadowOwnerUID     string
	TangWebSocketURL       string
	TangUsername           string
	TangPassword           string
	TangDeviceID           string
	BridgePeerID           string
	LeaderEnabled          bool
	LeaderKey              string
	LeaderInstance         string
	LeaderTTLSeconds       int
	EncryptionKey          string
}

func Load() Config {
	return Config{
		// Same vendor settings as this project's ChatProvider.java. Deployments
		// can supply their existing console configuration without changing code.
		TencentPushAndroidJSON: get("TENCENT_PUSH_ANDROID_JSON", `{"OPPOChannelID":"tuikit","HuaWeiCategory":"IM","VIVOCategory":"IM","HonorImportance":"NORMAL"}`),
		ListenAddr:             get("BRIDGE_LISTEN_ADDR", ":8090"),
		InternalSharedSecret:   get("BRIDGE_SHARED_SECRET", ""),
		StoreMode:              strings.ToLower(get("BRIDGE_STORE", "memory")),
		MySQLDSN:               get("BRIDGE_MYSQL_DSN", ""),
		AutoMigrate:            getBool("BRIDGE_AUTO_MIGRATE", false),
		RedisURL:               get("BRIDGE_REDIS_URL", ""),
		MaxAttempts:            getInt("BRIDGE_MAX_ATTEMPTS", 5),
		WorkerIntervalMS:       getInt("BRIDGE_WORKER_INTERVAL_MS", 250),
		TencentSDKAppID:        getIntFallback("TENCENT_SDK_APP_ID", "IM_SDK_APP_ID", 0),
		TencentSecretKey:       getFallback("TENCENT_SECRET_KEY", "IM_SDK_SECRET_KEY", ""),
		TencentAdminAccount:    getFallback("TENCENT_ADMIN_ACCOUNT", "IM_ADMIN_ACCOUNT", "administrator"),
		TencentRESTBaseURL:     get("TENCENT_REST_BASE_URL", "https://adminapisgp.im.qcloud.com"),
		TencentSigTTL:          getInt("TENCENT_USERSIG_TTL", 86400),
		TangBaseURL:            get("TANGSENG_BASE_URL", ""),
		TangToken:              get("TANGSENG_TOKEN", ""),
		TangUserEndpoint:       get("TANGSENG_USER_ENDPOINT", "/v1/user/usernameregister"),
		TangLoginEndpoint:      get("TANGSENG_LOGIN_ENDPOINT", "/v1/user/usernamelogin"),
		TangGroupEndpoint:      get("TANGSENG_GROUP_ENDPOINT", "/v1/group/create"),
		TangSendEndpoint:       get("TANGSENG_SEND_ENDPOINT", "/v1/message/send"),
		TangFetchEndpoint:      get("TANGSENG_FETCH_ENDPOINT", "/v1/bridge/messages/history"),
		TangShadowPassword:     get("TANGSENG_SHADOW_PASSWORD", ""),
		TangLegacyPassword:     get("TANGSENG_LEGACY_SHADOW_PASSWORD", ""),
		TangManufacturer:       get("TANGSENG_MANUFACTURER", "bridge"),
		TangInviteCode:         get("TANGSENG_INVITE_CODE", ""),
		TangShadowOwnerUID:     get("TANGSENG_SHADOW_OWNER_UID", ""),
		TangWebSocketURL:       get("TANGSENG_WS_URL", ""),
		TangUsername:           get("TANGSENG_USERNAME", ""),
		TangPassword:           get("TANGSENG_PASSWORD", ""),
		TangDeviceID:           get("TANGSENG_DEVICE_ID", ""),
		BridgePeerID:           get("BRIDGE_PEER_ID", ""),
		LeaderEnabled:          getBool("BRIDGE_LEADER_ENABLED", false),
		LeaderKey:              get("BRIDGE_LEADER_KEY", "tencent-tangseng-bridge:leader"),
		LeaderInstance:         get("BRIDGE_LEADER_INSTANCE", ""),
		LeaderTTLSeconds:       getInt("BRIDGE_LEADER_TTL_SECONDS", 15),
		EncryptionKey:          get("BRIDGE_ENCRYPTION_KEY", ""),
	}
}

func (c Config) Validate() error {
	if c.TencentPushAndroidJSON != "" {
		var settings map[string]any
		if json.Unmarshal([]byte(c.TencentPushAndroidJSON), &settings) != nil || settings == nil {
			return fmt.Errorf("TENCENT_PUSH_ANDROID_JSON must be a JSON object")
		}
	}
	if strings.TrimSpace(c.InternalSharedSecret) == "" {
		return fmt.Errorf("BRIDGE_SHARED_SECRET is required")
	}
	if c.MaxAttempts < 1 {
		return fmt.Errorf("BRIDGE_MAX_ATTEMPTS must be positive")
	}
	if c.WorkerIntervalMS < 10 {
		return fmt.Errorf("BRIDGE_WORKER_INTERVAL_MS must be at least 10")
	}
	if c.StoreMode == "mysql" && strings.TrimSpace(c.MySQLDSN) == "" {
		return fmt.Errorf("BRIDGE_MYSQL_DSN is required when BRIDGE_STORE=mysql")
	}
	return nil
}

func get(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value == 0 {
		return fallback
	}
	return value
}

func getFallback(primary, secondary, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(primary)); value != "" {
		return value
	}
	return get(secondary, fallback)
}

func getIntFallback(primary, secondary string, fallback int) int {
	if value := strings.TrimSpace(os.Getenv(primary)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed != 0 {
			return parsed
		}
	}
	return getInt(secondary, fallback)
}

func getBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}
