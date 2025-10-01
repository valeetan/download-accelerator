package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const FingerprintCookie = "fp"

func Fingerprint() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := c.Request.Cookie(FingerprintCookie); err != nil {
			fp := uuid.NewString()
			http.SetCookie(c.Writer, &http.Cookie{
				Name:     FingerprintCookie,
				Value:    fp,
				Path:     "/",
				HttpOnly: false,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   3600 * 24 * 365,
			})
		}
		c.Next()
	}
}
