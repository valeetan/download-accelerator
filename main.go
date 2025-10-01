package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"download-accelerator/internal/config"
	"download-accelerator/internal/controller"
	"download-accelerator/internal/dao"
	"download-accelerator/internal/middleware"
	"download-accelerator/internal/service"
)

func main() {
	cfg := config.Load()
	r := gin.Default()
	// 指纹中间件
	r.Use(middleware.Fingerprint())

	r.LoadHTMLGlob("web/templates/*.tmpl")
	r.Static("/static", "web/static")

	ls := service.NewLinkService(cfg.SigningKey, cfg.BaseURL)
	// 初始化 SQLite
	if err := os.MkdirAll("data", 0o755); err != nil {
		log.Println("mkdir data:", err)
	}
	if db, err := dao.OpenSQLite(cfg.DBPath); err == nil {
		ls.AttachDB(db)
		// 初始化管理员（仅在提供配置时执行，幂等）
		if cfg.AdminUser != "" && cfg.AdminPass != "" {
			_ = db.EnsureAdmin(context.Background(), cfg.AdminUser, sha256Hex(cfg.AdminPass))
		}
	} else {
		log.Println("open sqlite error:", err)
	}
	lc := controller.NewLinkController(ls, cfg)

	r.GET("/", lc.Index)
	r.POST("/", lc.Generate)
	r.GET("/d", lc.Proxy)

	// admin 登录与仪表盘
	if adb, err := dao.OpenSQLite(cfg.DBPath); err == nil {
		authc := controller.NewAdminAuthController(adb, cfg)
		r.GET("/admin/login", authc.LoginPage)
		r.POST("/admin/login", authc.LoginSubmit)

		adminAuth := middleware.NewAdminAuth(cfg.SigningKey + "/admin")
		admin := r.Group("/admin")
		admin.Use(adminAuth.Middleware())
		ac := controller.NewAdminController(adb)
		admin.GET("/", ac.Dashboard)
		admin.GET("/links", ac.Links)
		admin.POST("/logout", authc.Logout)
	}

	_ = r.Run(cfg.Addr)
}

// sha256Hex 用于初始化管理员口令存储
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
