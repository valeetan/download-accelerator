package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"path"
	"strconv"
	"strings"
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

// BuildAbsolute 规范化生成绝对URL：
// 1) 若 signedPath 已是绝对URL，则直接返回
// 2) 若 baseURL 存在，规范化拼接 baseURL + "/d?..."
func BuildAbsolute(baseURL, signedPath string) string {
	if isAbsURL(signedPath) {
		return signedPath
	}
	if baseURL == "" {
		return signedPath
	}
	bu, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + signedPath
	}
	sp, err := url.Parse(signedPath)
	if err != nil {
		return baseURL + signedPath
	}
	// 仅拼接路径，避免重复 host 片段
	bu.Path = strings.TrimRight(bu.Path, "/")
	if sp.Path != "" {
		bu.Path = path.Join(bu.Path, sp.Path)
	}
	bu.RawQuery = sp.RawQuery
	return bu.String()
}

func isAbsURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}
