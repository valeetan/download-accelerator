package controller

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"download-accelerator/internal/dao"
)

type AdminController struct {
	db *dao.DB
}

func NewAdminController(db *dao.DB) *AdminController { return &AdminController{db: db} }

func (a *AdminController) Dashboard(c *gin.Context) {
	ctx := context.Background()
	stat, _ := a.db.GetLinkStat(ctx)
	stats24h, _ := a.db.GetStats24h(ctx)
	topSources, _ := a.db.GetTopSources(ctx, 10)

	c.HTML(http.StatusOK, "admin_dashboard.tmpl", gin.H{
		"stat":       stat,
		"stats24h":   stats24h,
		"topSources": topSources,
	})
}

func (a *AdminController) Links(c *gin.Context) {
	ctx := context.Background()
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit := 20
	offset := (page - 1) * limit

	links, _ := a.db.GetLinks(ctx, limit, offset)

	c.HTML(http.StatusOK, "admin_links.tmpl", gin.H{
		"links": links,
		"page":  page,
	})
}
