package controller

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"download-accelerator/internal/config"
	"download-accelerator/internal/dao"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "source required"})
		return
	}
	if _, err := url.ParseRequestURI(source); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}
	// 探测源站元数据
	probe := service.NewProbeService(lc.cfg.HTTPTimeout)
	ctx, cancel := context.WithTimeout(c.Request.Context(), lc.cfg.HTTPTimeout)
	defer cancel()
	pr, _ := probe.Probe(ctx, source)

	exp := time.Now().Add(lc.cfg.TokenTTL)
	// 若未配置 BaseURL，则使用当前请求的 scheme://host
	if lc.cfg.BaseURL == "" {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		lc.cfg.BaseURL = scheme + "://" + c.Request.Host
	}
	absolute, err := lc.service.GenerateSignedLink(source, exp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign error"})
		return
	}
	if pr != nil && pr.OK {
		// 保存元数据
		_ = lc.service.SaveMeta(ctx, &dao.FileMeta{
			SourceURL: source,
			Filename:  pr.Filename,
			Size:      pr.Size,
			MimeType:  pr.MimeType,
			ETag:      pr.ETag,
			LastMod:   pr.LastMod,
		})
		// 保存链接记录（用于后台统计/限制）
		if fp, _ := c.Cookie("fp"); fp != "" {
			_ = lc.service.RecordLink(ctx, source, absolute, fp, exp)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":   pr != nil && pr.OK,
		"link": absolute,
		"meta": gin.H{
			"filename": pr.Filename,
			"size":     pr.Size,
			"mimeType": pr.MimeType,
		},
	})
}

func (lc *LinkController) Proxy(c *gin.Context) {
	valid := lc.service.VerifyQuery(c.Request.URL.Query())
	if !valid {
		c.String(http.StatusForbidden, "invalid or expired link")
		return
	}
	source := c.Query("u")
	p := service.NewProxyService(lc.cfg.HTTPTimeout, lc.cfg.MaxConcurrent)
	p.SetThrottle(lc.cfg.ThrottleBps)

	// 查询 links 记录
	var linkID int64
	if s := c.Request.URL; s != nil {
		// 用路径+查询作为键
		if lc.service != nil && lc.service.DB() != nil {
			if id, _, fp, _, err := lc.service.DB().GetLinkBySignedPath(c, s.Path+"?"+s.RawQuery); err == nil {
				linkID = id
				// 可疑分享：cookie 指纹与生成指纹不一致
				if cfp, _ := c.Cookie("fp"); cfp != "" && fp != "" && cfp != fp {
					_ = lc.service.DB().InsertLinkEvent(c, linkID, c.ClientIP(), c.Request.UserAgent(), cfp, 0, true)
				}
			}
		}
	}

	n, _ := p.ProxyWithCount(c.Request.Context(), source, c.Writer, c.Request)
	if linkID > 0 && lc.service != nil && lc.service.DB() != nil {
		cfp, _ := c.Cookie("fp")
		_ = lc.service.DB().InsertLinkEvent(c, linkID, c.ClientIP(), c.Request.UserAgent(), cfp, n, false)
	}
}
