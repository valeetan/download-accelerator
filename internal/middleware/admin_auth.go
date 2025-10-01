package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const adminCookie = "admin_session"

type AdminAuth struct {
	secret []byte
}

func NewAdminAuth(secret string) *AdminAuth { return &AdminAuth{secret: []byte(secret)} }

func (a *AdminAuth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.Verify(c.Request) {
			c.Next()
			return
		}
		// 未登录：重定向到前台首页
		c.Redirect(http.StatusFound, "/")
		c.Abort()
	}
}

func (a *AdminAuth) Sign(username string, exp time.Time) string {
	payload := username + "|" + exp.UTC().Format(time.RFC3339)
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "|" + sig
}

func (a *AdminAuth) Verify(r *http.Request) bool {
	c, err := r.Cookie(adminCookie)
	if err != nil || c.Value == "" {
		return false
	}
	parts := []byte(c.Value)
	// simple parse: payload|sig with two '|'
	// find last '|'
	last := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '|' {
			last = i
			break
		}
	}
	if last <= 0 {
		return false
	}
	payload := string(parts[:last])
	sig := string(parts[last+1:])
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return false
	}
	// check exp
	// payload is username|exp
	p2 := payload
	// find last '|'
	last2 := -1
	for i := len(p2) - 1; i >= 0; i-- {
		if p2[i] == '|' {
			last2 = i
			break
		}
	}
	if last2 <= 0 {
		return false
	}
	expStr := p2[last2+1:]
	if t, err := time.Parse(time.RFC3339, expStr); err == nil {
		if time.Now().After(t) {
			return false
		}
	}
	return true
}

func (a *AdminAuth) SetSession(w http.ResponseWriter, username string, ttl time.Duration) {
	exp := time.Now().Add(ttl)
	token := a.Sign(username, exp)
	http.SetCookie(w, &http.Cookie{
		Name: adminCookie, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Expires: exp,
	})
}

func (a *AdminAuth) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: adminCookie, Value: "", Path: "/", MaxAge: -1})
}
