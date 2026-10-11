package media

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// SHA256 returns the stable media cache key used by media_cache. Download and
// object-storage adapters can use this helper without exposing credentials to
// the message relay.
func SHA256(r io.Reader) (string, int64, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
