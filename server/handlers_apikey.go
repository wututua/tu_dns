package server

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func (a *App) handleListApiKeys(c *gin.Context) {
	items, err := a.apikey.List(CurrentUserID(c))
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, items)
}

func (a *App) handleCreateApiKey(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		BadRequest(c, "名称必填")
		return
	}
	raw, key, err := a.apikey.Create(CurrentUserID(c), req.Name)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"raw_key": raw, "key": key})
}

func (a *App) handleRevokeApiKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := a.apikey.Revoke(CurrentUserID(c), uint(id)); err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}

func (a *App) handleAdminListApiKeys(c *gin.Context) {
	items, err := a.apikey.List(0)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, items)
}

func (a *App) handleAdminRevokeApiKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := a.apikey.Revoke(0, uint(id)); err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}
