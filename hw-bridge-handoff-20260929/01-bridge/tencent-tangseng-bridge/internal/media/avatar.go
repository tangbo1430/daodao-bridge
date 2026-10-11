package media

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/domain"
	"github.com/qicaihui/tencent-tangseng-bridge/internal/store"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var ErrGroupAvatarNotFound = errors.New("group avatar not available")

func (s *Service) AvatarURL(ctx context.Context, kind, id string) (string, error) {
	if kind != "users" && kind != "groups" {
		return "", errors.New("avatar type invalid")
	}
	object, err := s.convertObject(ctx, domain.MessagePart{Kind: "image", Name: "avatar.png", URL: "/v1/" + kind + "/" + url.PathEscape(id) + "/avatar"}, "", "", 1, false)
	if err != nil {
		var response *downloadHTTPError
		if kind == "groups" && errors.As(err, &response) && response.status == http.StatusNotFound {
			return "", ErrGroupAvatarNotFound
		}
		return "", err
	}
	return s.PublicURL(object.SHA256), nil
}
func (s *Service) UploadAvatar(ctx context.Context, actor, group, source string) error {
	if s.Token == nil {
		return errors.New("avatar actor unavailable")
	}
	token, ok := s.Token(actor)
	if !ok || token == "" {
		return errors.New("avatar actor not logged in")
	}
	input, err := s.download(ctx, source)
	if err != nil {
		return err
	}
	defer input.Cleanup()
	key := "avatar-upload:" + domain.StableID(actor, group)
	if s.Store != nil {
		previous, e := s.Store.GetCursor(ctx, key)
		if e != nil && !errors.Is(e, store.ErrNotFound) {
			return e
		}
		if previous.Aux == input.Hash {
			return nil
		}
	}
	firstUserAvatar := false
	if group == "" {
		// The native service emits userAvatarUpdate BEFORE persisting its first
		// is_upload_avatar=1 flag. A fast recipient can cache the default image.
		// Only that first upload needs one further native notification after the
		// successful response. Use the owner's normal API, never forge a CMD.
		firstUserAvatar, err = s.isFirstUserAvatar(ctx, actor, token)
		if err != nil {
			return err
		}
	}
	config, _, err := image.DecodeConfig(input)
	if err != nil || config.Width < 1 || config.Height < 1 || int64(config.Width)*int64(config.Height) > 20_000_000 {
		return errors.New("avatar image invalid or too large")
	}
	_, _ = input.Seek(0, 0)
	decoded, _, err := image.Decode(input)
	if err != nil {
		return errors.New("avatar image invalid")
	}
	tmp, err := os.CreateTemp("", "hwbridge-avatar-*")
	if err != nil {
		return err
	}
	defer func() { tmp.Close(); os.Remove(tmp.Name()) }()
	writer := multipart.NewWriter(tmp)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		return err
	}
	if err = png.Encode(part, decoded); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	size, _ := tmp.Seek(0, io.SeekCurrent)
	_, _ = tmp.Seek(0, 0)
	endpoint := "/v1/users/" + url.PathEscape(actor) + "/avatar"
	if group != "" {
		endpoint = "/v1/groups/" + url.PathEscape(group) + "/avatar"
	}
	upload := func() error {
		if _, err := tmp.Seek(0, io.SeekStart); err != nil {
			return err
		}
		req, _ := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(s.TangBase, "/")+endpoint, io.NopCloser(tmp))
		req.ContentLength = size
		req.Header.Set("token", token)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		client := s.client()
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		response, err := client.Do(req)
		if err != nil {
			return errors.New("avatar upload failed")
		}
		defer response.Body.Close()
		if response.StatusCode != 200 {
			return errors.New("avatar upload rejected")
		}
		var result struct {
			Status int `json:"status"`
			Code   int `json:"code"`
		}
		if err = json.NewDecoder(io.LimitReader(response.Body, 8192)).Decode(&result); err != nil && err != io.EOF {
			return errors.New("avatar upload response invalid")
		}
		if result.Status >= 400 || (result.Code != 0 && result.Code != 200) {
			return errors.New("avatar upload business rejection")
		}
		return nil
	}
	if err = upload(); err != nil {
		return err
	}
	if firstUserAvatar {
		if err = upload(); err != nil {
			return err
		}
	}
	if s.Store != nil {
		return s.Store.PutCursor(ctx, domain.SyncCursor{Key: key, Aux: input.Hash, Value: time.Now().Unix(), StartedAt: time.Now().Unix()})
	}
	return nil
}

func (s *Service) isFirstUserAvatar(ctx context.Context, actor, token string) (bool, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.TangBase, "/")+"/v1/users/"+url.PathEscape(actor), nil)
	req.Header.Set("token", token)
	client := s.client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(req)
	if err != nil {
		return false, errors.New("avatar state query failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, errors.New("avatar state query rejected")
	}
	var profile struct {
		UID      string `json:"uid"`
		Uploaded *int   `json:"is_upload_avatar"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 128<<10)).Decode(&profile) != nil || profile.UID != actor {
		return false, errors.New("avatar state response invalid")
	}
	return profile.Uploaded != nil && *profile.Uploaded == 0, nil
}
