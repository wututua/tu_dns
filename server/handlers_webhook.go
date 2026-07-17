package server

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func (a *App) handleAdminWebhooks(c *gin.Context) {
	items, err := a.webhook.List()
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, items)
}

func (a *App) handleAdminCreateWebhook(c *gin.Context) {
	var req struct {
		Name   string `json:"name"`
		URL    string `json:"url"`
		Events string `json:"events"`
		Secret string `json:"secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	if req.Name == "" || req.URL == "" || req.Events == "" {
		BadRequest(c, "名称、URL和事件必填")
		return
	}
	w, err := a.webhook.Create(req.Name, req.URL, req.Events, req.Secret)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, w)
}

func (a *App) handleAdminUpdateWebhook(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Events  string `json:"events"`
		Secret  string `json:"secret"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	w, err := a.webhook.Update(uint(id), req.Name, req.URL, req.Events, req.Secret, req.Enabled)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, w)
}

func (a *App) handleAdminDeleteWebhook(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := a.webhook.Delete(uint(id)); err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}
