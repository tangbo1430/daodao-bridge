package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"time"
)

// ServeSource gives the PHP gateway a short-lived, authenticated public object
// location. Binary data can then use the existing COS/CDN route without passing
// through the Bridge CVM's public egress. It never sends TangSeng user tokens.
func (s *Service) ServeSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	id := r.URL.Query().Get("id")
	if !s.Verify(id, r.URL.Query().Get("sig")) {
		http.Error(w, "invalid media signature", 403)
		return
	}
	object, err := s.Store.GetMedia(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	address, err := s.resolve(object.ObjectURL)
	if err != nil {
		http.Error(w, "media unavailable", 502)
		return
	}
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, address, nil)
	req.Header.Set("Range", "bytes=0-0")
	resp, err := s.client().Do(req)
	if err != nil {
		http.Error(w, "media unavailable", 502)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		http.Error(w, "media unavailable", 502)
		return
	}
	u := resp.Request.URL
	// An internal avatar endpoint may return bytes itself instead of redirecting.
	// In that case PHP continues using the bounded Bridge streaming endpoint.
	ips, err := net.DefaultResolver.LookupIPAddr(r.Context(), u.Hostname())
	public := err == nil && len(ips) > 0 && u.Scheme == "https" && effectivePort(u) == "443" && u.User == nil
	addresses := make([]string, 0, len(ips))
	for _, ip := range ips {
		public = public && publicIP(ip.IP)
		addresses = append(addresses, ip.IP.String())
	}
	payload := map[string]any{"id": id, "expires": time.Now().Add(time.Minute).Unix(), "mode": "bridge"}
	if public {
		payload["mode"], payload["url"], payload["addresses"] = "public", u.String(), addresses
		payload["name"], payload["mime"], payload["size"] = object.Name, object.MIME, object.Size
	}
	body, _ := json.Marshal(payload)
	mac := hmac.New(sha256.New, []byte(s.Secret))
	mac.Write([]byte("hw-bridge-media-source-v1\n"))
	mac.Write(body)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]string{"payload": string(body), "signature": hex.EncodeToString(mac.Sum(nil))})
}
