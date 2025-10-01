package controller

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"download-accelerator/internal/config"
	"download-accelerator/internal/service"
)

type LinkController struct {
	service *service.LinkService
	cfg     *config.Config
}

func NewLinkController(s *service.LinkService, cfg *config.Config) *LinkController {
	return &LinkController{service: s, cfg: cfg}
}

func (lc *LinkController) Index(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.HTML(http.StatusOK, "index.tmpl", gin.H{})
}

func (lc *LinkController) Generate(c *gin.Context) {
	source := c.PostForm("source")
	if source == "" {
		c.String(http.StatusBadRequest, "source required")
		return
	}
	if _, err := url.ParseRequestURI(source); err != nil {
		c.String(http.StatusBadRequest, "invalid url")
		return
	}

	exp := time.Now().Add(lc.cfg.TokenTTL)
	absolute, err := lc.service.GenerateSignedLink(source, exp)
	if err != nil {
		c.String(http.StatusInternalServerError, "sign error")
		return
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, absolute)
}

func (lc *LinkController) Proxy(c *gin.Context) {
	valid := lc.service.VerifyQuery(c.Request.URL.Query())
	if !valid {
		c.String(http.StatusForbidden, "invalid or expired link")
		return
	}
	// TODO: 实现源站代理转发与缓存控制
	c.String(http.StatusNotImplemented, "proxy not implemented yet")
}
