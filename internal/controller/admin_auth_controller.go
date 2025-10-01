package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"download-accelerator/internal/config"
	"download-accelerator/internal/dao"
	"download-accelerator/internal/middleware"
)

type AdminAuthController struct {
	db   *dao.DB
	auth *middleware.AdminAuth
	cfg  *config.Config
}

func NewAdminAuthController(db *dao.DB, cfg *config.Config) *AdminAuthController {
	return &AdminAuthController{db: db, auth: middleware.NewAdminAuth(cfg.SigningKey + "/admin"), cfg: cfg}
}

func (a *AdminAuthController) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_login.tmpl", gin.H{"err": ""})
}

func (a *AdminAuthController) LoginSubmit(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	if username == "" || password == "" {
		c.HTML(http.StatusBadRequest, "admin_login.tmpl", gin.H{"err": "请输入用户名和密码"})
		return
	}
	// bootstrap admin if not exists
	if a.cfg.AdminUser != "" && a.cfg.AdminPass != "" {
		_ = a.db.EnsureAdmin(context.Background(), a.cfg.AdminUser, sha256Hex(a.cfg.AdminPass))
	}
	id, passhash, err := a.db.GetUserByName(context.Background(), username)
	if err != nil || id == 0 {
		c.HTML(http.StatusUnauthorized, "admin_login.tmpl", gin.H{"err": "用户不存在"})
		return
	}
	if passhash != sha256Hex(password) {
		c.HTML(http.StatusUnauthorized, "admin_login.tmpl", gin.H{"err": "密码错误"})
		return
	}
	a.auth.SetSession(c.Writer, username, 24*time.Hour)
	c.Redirect(http.StatusFound, "/admin")
}

func (a *AdminAuthController) Logout(c *gin.Context) {
	a.auth.ClearSession(c.Writer)
	c.Redirect(http.StatusFound, "/admin/login")
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
