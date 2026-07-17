package server

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func (a *App) handleListNotifications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	items, total, err := a.notification.List(CurrentUserID(c), page, 20)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"items": items, "total": total, "page": page})
}

func (a *App) handleUnreadCount(c *gin.Context) {
	count, err := a.notification.UnreadCount(CurrentUserID(c))
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"count": count})
}

func (a *App) handleMarkRead(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := a.notification.MarkRead(CurrentUserID(c), uint(id)); err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}

func (a *App) handleMarkAllRead(c *gin.Context) {
	if err := a.notification.MarkAllRead(CurrentUserID(c)); err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}
