package media

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const MaxSize int64 = 100 << 20

type downloadHTTPError struct{ status int }

func (e *downloadHTTPError) Error() string { return fmt.Sprintf("media download HTTP %d", e.status) }

// The PHP gateway crosses the public network. Its backpressure must not inherit
// the short timeout used for ingesting objects inside the TangSeng network.
const streamTimeout = 10 * time.Minute

var objectID = regexp.MustCompile("^[a-f0-9]{64}$")

type Service struct {
	Store                                         store.Store
	TangBase, PublicBase, Secret, FFmpeg, FFprobe string
	Token                                         func(string) (string, bool)
}
type fileData struct {
	*os.File
	Size       int64
	MIME, Hash string
}

func (f *fileData) Cleanup() { f.Close(); os.Remove(f.Name()) }
func (s *Service) signature(id string) string {
	h := hmac.New(sha256.New, []byte(s.Secret))
	h.Write([]byte("hw-bridge-media-v1\n" + id))
	return hex.EncodeToString(h.Sum(nil))
}
func (s *Service) PublicURL(id string) string {
	return strings.TrimRight(s.PublicBase, "/") + "?id=" + id + "&sig=" + s.signature(id)
}
func (s *Service) Verify(id, sig string) bool {
	return objectID.MatchString(id) && s.Secret != "" && hmac.Equal([]byte(sig), []byte(s.signature(id)))
}
func (s *Service) resolve(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Fragment != "" {
		return "", errors.New("media URL invalid")
	}
	if !u.IsAbs() {
		p := strings.TrimPrefix(u.Path, "/")
		p = strings.TrimPrefix(p, "v1/")
		if !allowedTangPath("/v1/" + p) {
			return "", errors.New("media path not allowed")
		}
		return strings.TrimRight(s.TangBase, "/") + "/v1/" + p, nil
	}
	if err = s.checkURL(u); err != nil {
		return "", err
	}
	return u.String(), nil
}
func allowedTangPath(p string) bool {
	if strings.Contains(p, "..") || strings.ContainsAny(p, "\\\x00") {
		return false
	}
	if strings.HasPrefix(p, "/v1/file/preview/") {
		return true
	}
	a := strings.Split(strings.Trim(p, "/"), "/")
	return len(a) == 4 && a[0] == "v1" && (a[1] == "users" || a[1] == "groups") && a[2] != "" && a[3] == "avatar"
}
func (s *Service) checkURL(u *url.URL) error {
	if u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("media scheme not allowed")
	}
	base, _ := url.Parse(s.TangBase)
	if base != nil && u.Host == base.Host && u.Scheme == base.Scheme {
		if !allowedTangPath(u.Path) {
			return errors.New("TangSeng media path not allowed")
		}
		return nil
	}
	if u.Port() != "" && u.Port() != "443" && u.Port() != "80" {
		return errors.New("media port not allowed")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !publicIP(ip) {
		return errors.New("private media address rejected")
	}
	return nil
}
func publicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast()
}
func (s *Service) client() *http.Client {
	base, _ := url.Parse(s.TangBase)
	transport := &http.Transport{TLSHandshakeTimeout: 8 * time.Second, ResponseHeaderTimeout: 15 * time.Second, DisableKeepAlives: true}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		allowed := base != nil && host == base.Hostname() && port == effectivePort(base)
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, errors.New("media DNS failed")
		}
		for _, v := range ips {
			if !allowed && !publicIP(v.IP) {
				return nil, errors.New("private media address rejected")
			}
		}
		for _, v := range ips {
			conn, e := (&net.Dialer{Timeout: 8 * time.Second}).DialContext(ctx, network, net.JoinHostPort(v.IP.String(), port))
			if e == nil {
				return conn, nil
			}
		}
		return nil, errors.New("media connection failed")
	}
	return &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("media redirect limit")
		}
		return s.checkURL(req.URL)
	}}
}
func effectivePort(u *url.URL) string {
	if u.Port() != "" {
		return u.Port()
	}
	if u.Scheme == "https" {
		return "443"
	}
	return "80"
}
func (s *Service) download(ctx context.Context, raw string) (*fileData, error) {
	address, err := s.resolve(raw)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", address, nil)
	if err != nil {
		return nil, err
	}
	response, err := s.client().Do(req)
	if err != nil {
		return nil, errors.New("media download failed")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, &downloadHTTPError{status: response.StatusCode}
	}
	if response.ContentLength > MaxSize {
		return nil, errors.New("media exceeds 100 MB")
	}
	f, err := os.CreateTemp("", "hwbridge-media-*")
	if err != nil {
		return nil, err
	}
	d := &fileData{File: f, MIME: response.Header.Get("Content-Type")}
	h := sha256.New()
	d.Size, err = io.Copy(io.MultiWriter(f, h), io.LimitReader(response.Body, MaxSize+1))
	if err != nil || d.Size > MaxSize {
		d.Cleanup()
		return nil, errors.New("media download incomplete or too large")
	}
	d.Hash = hex.EncodeToString(h.Sum(nil))
	_, err = f.Seek(0, 0)
	if err != nil {
		d.Cleanup()
		return nil, err
	}
	if d.MIME == "" {
		var head [512]byte
		n, _ := f.Read(head[:])
		d.MIME = http.DetectContentType(head[:n])
		_, _ = f.Seek(0, 0)
	}
	return d, nil
}
func (s *Service) upload(ctx context.Context, actor, channel string, channelType int, d *fileData, extension string) (string, error) {
	if s.Token == nil {
		return "", errors.New("media actor unavailable")
	}
	token, ok := s.Token(actor)
	if !ok || token == "" {
		return "", errors.New("media actor not logged in")
	}
	if strings.ContainsAny(channel, "/\\") || channel == "" {
		return "", errors.New("media channel invalid")
	}
	// The hash makes a retry reuse the same object rather than leave random duplicates.
	target := fmt.Sprintf("/%d/%s/%s%s", channelType, channel, d.Hash, extension)
	query := url.Values{"type": {"chat"}, "path": {target}}
	tmp, err := os.CreateTemp("", "hwbridge-upload-*")
	if err != nil {
		return "", err
	}
	defer func() { tmp.Close(); os.Remove(tmp.Name()) }()
	writer := multipart.NewWriter(tmp)
	part, err := writer.CreateFormFile("file", d.Hash+extension)
	if err != nil {
		return "", err
	}
	_, _ = d.Seek(0, 0)
	if _, err = io.Copy(part, d); err != nil {
		return "", err
	}
	_ = writer.WriteField("contenttype", d.MIME)
	if err = writer.Close(); err != nil {
		return "", err
	}
	size, _ := tmp.Seek(0, io.SeekCurrent)
	_, _ = tmp.Seek(0, 0)
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(s.TangBase, "/")+"/v1/file/upload?"+query.Encode(), tmp)
	if err != nil {
		return "", err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("token", token)
	// Upload never follows redirects and never forwards credentials to object storage.
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("media upload failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("media upload HTTP %d", resp.StatusCode)
	}
	var out struct {
		Path string `json:"path"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out) != nil || !strings.HasPrefix(out.Path, "file/preview/") {
		return "", errors.New("media upload contract mismatch")
	}
	return out.Path, nil
}
func (s *Service) audio(ctx context.Context, input *fileData, toTang bool) (*fileData, error) {
	exe := s.FFmpeg
	if exe == "" {
		exe = "ffmpeg"
	}
	ext := ".m4a"
	codec := "aac"
	rate := "16000"
	bitrate := "32k"
	mimeType := "audio/mp4"
	if toTang {
		ext = ".amr"
		codec = "libopencore_amrnb"
		rate = "8000"
		bitrate = "12.2k"
		mimeType = "audio/amr"
	}
	tmp, err := os.CreateTemp("", "hwbridge-audio-*"+ext)
	if err != nil {
		return nil, err
	}
	output := tmp.Name()
	tmp.Close()
	defer os.Remove(output)
	work, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	args := []string{"-nostdin", "-v", "error", "-y", "-protocol_whitelist", "file,pipe", "-format_whitelist", "mov,mp3,amr,wav,ogg,flac,matroska,webm", "-i", input.Name(), "-vn", "-ac", "1", "-ar", rate, "-c:a", codec, "-b:a", bitrate, output}
	if err = exec.CommandContext(work, exe, args...).Run(); err != nil {
		return nil, errors.New("audio conversion failed")
	}
	f, err := os.Open(output)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	copyFile, err := os.CreateTemp("", "hwbridge-converted-*")
	if err != nil {
		return nil, err
	}
	d := &fileData{File: copyFile, MIME: mimeType}
	h := sha256.New()
	d.Size, err = io.Copy(io.MultiWriter(copyFile, h), io.LimitReader(f, MaxSize+1))
	if err != nil || d.Size > MaxSize {
		d.Cleanup()
		return nil, errors.New("converted audio too large")
	}
	d.Hash = hex.EncodeToString(h.Sum(nil))
	_, _ = d.Seek(0, 0)
	return d, nil
}
func mediaExtension(p domain.MessagePart, mimeType string, toTang bool) string {
	if p.Kind == "audio" {
		if toTang {
			return ".amr"
		}
		return ".m4a"
	}
	if p.Kind == "image" {
		if p.Format == 2 {
			return ".gif"
		}
		if strings.Contains(mimeType, "png") {
			return ".png"
		}
		return ".jpg"
	}
	if p.Kind == "video" {
		return ".mp4"
	}
	ext := strings.ToLower(filepath.Ext(p.Name))
	if len(ext) > 12 || strings.ContainsAny(ext, "/\\") {
		return ".bin"
	}
	if ext == "" {
		return ".bin"
	}
	return ext
}
func (s *Service) convertObject(ctx context.Context, p domain.MessagePart, actor, channel string, channelType int, toTang bool) (domain.MediaObject, error) {
	d, err := s.download(ctx, p.URL)
	if err != nil {
		return domain.MediaObject{}, err
	}
	defer d.Cleanup()
	if p.Kind == "audio" {
		d2, e := s.audio(ctx, d, toTang)
		if e != nil {
			return domain.MediaObject{}, e
		}
		defer d2.Cleanup()
		d = d2
	}
	uploadRequired := toTang || p.Kind == "audio"
	uploadKey := "media-upload:" + domain.StableID(d.Hash, actor, channel, strconv.Itoa(channelType), mediaExtension(p, d.MIME, toTang))
	if uploadRequired {
		cached, e := s.Store.GetCursor(ctx, uploadKey)
		if e == nil {
			var v domain.MediaObject
			if e = json.Unmarshal([]byte(cached.Aux), &v); e != nil {
				return v, e
			}
			return v, nil
		}
		if !errors.Is(e, store.ErrNotFound) {
			return domain.MediaObject{}, e
		}
	} else if previous, e := s.Store.GetMedia(ctx, d.Hash); e == nil {
		return previous, nil
	} else if !errors.Is(e, store.ErrNotFound) {
		return domain.MediaObject{}, e
	}
	objectURL := p.URL
	if uploadRequired {
		objectURL, err = s.upload(ctx, actor, channel, channelType, d, mediaExtension(p, d.MIME, toTang))
		if err != nil {
			return domain.MediaObject{}, err
		}
	}
	v := domain.MediaObject{SHA256: d.Hash, SourceURL: p.URL, ObjectURL: objectURL, MIME: d.MIME, Size: d.Size, Name: path.Base(strings.ReplaceAll(p.Name, "\\", "/"))}
	if v.Name == "" || v.Name == "." {
		v.Name = d.Hash + mediaExtension(p, d.MIME, toTang)
	}
	// Keep existing viewer capabilities stable. Upload reuse is separately
	// scoped by actor and channel, so a prior download cannot bypass upload.
	if _, e := s.Store.GetMedia(ctx, v.SHA256); errors.Is(e, store.ErrNotFound) {
		if err = s.Store.PutMedia(ctx, v); err != nil {
			return v, err
		}
	} else if e != nil {
		return v, e
	}
	if uploadRequired {
		b, e := json.Marshal(v)
		if e != nil {
			return v, e
		}
		if e = s.Store.PutCursor(ctx, domain.SyncCursor{Key: uploadKey, Aux: string(b), Value: time.Now().Unix()}); e != nil {
			return v, e
		}
	}
	return v, nil
}
func (s *Service) Prepare(ctx context.Context, event domain.MessageEvent, from, to domain.Identity) (domain.MessageEvent, error) {
	if len(event.Parts) == 0 {
		return event, nil
	}
	event.Parts = append([]domain.MessagePart(nil), event.Parts...)
	toTang := event.Source == domain.SourceTencent
	actor, channel := event.ToAccount, event.ConversationID
	if toTang {
		actor = from.TargetID
		channel = to.TargetID
	}
	ct := 1
	if event.ConversationType == domain.ConversationGroup {
		ct = 2
	}
	for i, p := range event.Parts {
		if p.Kind == "text" || p.Kind == "tip" || p.Kind == "card" {
			continue
		}
		if s.PublicBase == "" {
			return event, errors.New("media public gateway not configured")
		}
		obj, err := s.convertObject(ctx, p, actor, channel, ct, toTang)
		if err != nil {
			return event, err
		}
		p.Size = obj.Size
		if toTang {
			p.URL = obj.ObjectURL
		} else {
			p.URL = s.PublicURL(obj.SHA256)
		}
		if p.Name == "" {
			p.Name = obj.Name
		}
		if p.Kind == "image" && p.Format != 2 && strings.Contains(obj.MIME, "png") {
			p.Format = 3
		}
		if p.Kind == "video" {
			if p.Cover == "" {
				return event, errors.New("video thumbnail missing")
			}
			cover, e := s.convertObject(ctx, domain.MessagePart{Kind: "image", URL: p.Cover}, actor, channel, ct, toTang)
			if e != nil {
				return event, e
			}
			p.CoverSize = cover.Size
			if toTang {
				p.Cover = cover.ObjectURL
			} else {
				p.Cover = s.PublicURL(cover.SHA256)
			}
		}
		event.Parts[i] = p
	}
	return event, nil
}

// ServeHTTP is mounted behind the existing internal HMAC middleware. The APK
// sees only the signed PHP URL; TangSeng URLs and credentials stay server-side.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "method not allowed", 405)
		return
	}
	id, sig := r.URL.Query().Get("id"), r.URL.Query().Get("sig")
	if !s.Verify(id, sig) {
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
	// TangSeng preview routes implement GET only; HEAD is answered from persisted metadata.
	w.Header().Set("Content-Type", object.MIME)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	disposition := "inline"
	if !strings.HasPrefix(object.MIME, "image/") && !strings.HasPrefix(object.MIME, "audio/") && !strings.HasPrefix(object.MIME, "video/") {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": object.Name}))
	if r.Method == "HEAD" {
		w.Header().Set("Content-Length", strconv.FormatInt(object.Size, 10))
		return
	}
	// Override the business API write deadline only for this bounded media stream.
	// ResponseRecorder and middleware without deadline support return ErrNotSupported.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(streamTimeout))
	req, _ := http.NewRequestWithContext(r.Context(), "GET", address, nil)
	if v := r.Header.Get("Range"); v != "" {
		if !regexp.MustCompile("^bytes=[0-9]*-[0-9]*$").MatchString(v) {
			http.Error(w, "invalid range", 416)
			return
		}
		req.Header.Set("Range", v)
	}
	client := s.client()
	client.Timeout = streamTimeout
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "media unavailable", 502)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 206 && resp.StatusCode != 416 {
		http.Error(w, "media unavailable", 502)
		return
	}
	if resp.ContentLength > MaxSize {
		http.Error(w, "media too large", 413)
		return
	}
	for _, k := range []string{"Content-Length", "Content-Range", "ETag", "Last-Modified"} {
		if v := resp.Header.Get(k); v != "" {
			w.Header().Set(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_ = http.NewResponseController(w).Flush()
	if _, err := io.Copy(w, io.LimitReader(resp.Body, MaxSize)); err != nil {
		// Never finish a truncated resource as a successful complete HTTP response.
		panic(http.ErrAbortHandler)
	}
}
