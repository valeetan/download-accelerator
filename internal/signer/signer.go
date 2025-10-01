package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type LinkSigner struct {
	key []byte
}

func NewLinkSigner(secret string) *LinkSigner {
	return &LinkSigner{key: []byte(secret)}
}

// SignURL 生成带过期时间与签名的相对路径，如 /d?u=...&exp=...&sig=...
func (s *LinkSigner) SignURL(source string, expiry time.Time) (string, error) {
	q := url.Values{}
	q.Set("u", source)
	q.Set("exp", strconv.FormatInt(expiry.Unix(), 10))

	sig := s.sign(q.Encode())
	q.Set("sig", sig)

	return "/d?" + q.Encode(), nil
}

func (s *LinkSigner) Verify(values url.Values) bool {
	u := values.Get("u")
	exp := values.Get("exp")
	sig := values.Get("sig")
	if u == "" || exp == "" || sig == "" {
		return false
	}

	// 过期校验
	unix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || time.Now().Unix() > unix {
		return false
	}

	// 签名校验
	q := url.Values{}
	q.Set("u", u)
	q.Set("exp", exp)
	expected := s.sign(q.Encode())
	return hmac.Equal([]byte(sig), []byte(expected))
}

func (s *LinkSigner) sign(payload string) string {
	h := hmac.New(sha256.New, s.key)
	h.Write([]byte(payload))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h.Sum(nil))
}

// BuildAbsolute 根据 BaseURL 拼接对外可访问链接
func BuildAbsolute(baseURL, signedPath string) string {
	if baseURL == "" {
		return signedPath
	}
	return fmt.Sprintf("%s%s", stringsTrimRightSlash(baseURL), signedPath)
}

func stringsTrimRightSlash(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}
