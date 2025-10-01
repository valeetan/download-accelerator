package main

import (
	"github.com/gin-gonic/gin"

	"download-accelerator/internal/config"
	"download-accelerator/internal/controller"
	"download-accelerator/internal/service"
)

func main() {
	cfg := config.Load()
	r := gin.Default()

	r.LoadHTMLGlob("web/templates/*.tmpl")
	r.Static("/static", "web/static")

	ls := service.NewLinkService(cfg.SigningKey, cfg.BaseURL)
	lc := controller.NewLinkController(ls, cfg)

	r.GET("/", lc.Index)
	r.POST("/", lc.Generate)
	r.GET("/d", lc.Proxy)

	_ = r.Run(cfg.Addr)
}
