package account

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func PHPLookup(endpoint, secret string) func(context.Context, string, string) (string, error) {
	return func(ctx context.Context, username, reserve string) (string, error) {
		return phpRequest(ctx, endpoint, secret, map[string]string{"username": username, "reserve_id": reserve})
	}
}
func PHPStatus(endpoint, secret string) func(context.Context, string, string, int64) error {
	return func(ctx context.Context, tim, state string, version int64) error {
		_, err := phpRequest(ctx, endpoint, secret, map[string]any{"tim_uid": tim, "state": state, "reason": state, "version": version})
		return err
	}
}
func phpRequest(ctx context.Context, endpoint, secret string, payload any) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || secret == "" {
		return "", fmt.Errorf("secure PHP lookup configuration missing")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	b := make([]byte, 16)
	if _, err = rand.Read(b); err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(b)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(ts + "\n" + nonce + "\n"))
	h.Write(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("PHP lookup request invalid")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-HWB-Timestamp", ts)
	req.Header.Set("X-HWB-Nonce", nonce)
	req.Header.Set("X-HWB-Sign", hex.EncodeToString(h.Sum(nil)))
	client := &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("PHP registration lookup unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("PHP lookup HTTP %d", resp.StatusCode)
	}
	var r struct {
		Code *int   `json:"code"`
		UID  string `json:"tim_uid"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&r) != nil || r.Code == nil || *r.Code != 0 {
		return "", fmt.Errorf("PHP lookup response invalid")
	}
	return r.UID, nil
}
