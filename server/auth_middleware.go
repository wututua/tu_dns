package server

import (
	"strings"

	"tudns/auth"
	"tudns/config"
	"tudns/db"
	"tudns/models"

	"github.com/gin-gonic/gin"
)

const (
	CtxUserID   = "user_id"
	CtxUsername = "username"
	CtxRole     = "role"
)

func BearerAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		gdb := db.Get()
		if gdb == nil {
			Fail(c, 503, 503, "系统未安装")
			c.Abort()
			return
		}
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			Unauthorized(c, "未登录")
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		claims, err := auth.ParseToken(cfg.Security.SecretKey, token)
		if err != nil {
			Unauthorized(c, "登录已失效")
			c.Abort()
			return
		}
		var u models.User
		if err := gdb.First(&u, claims.UserID).Error; err != nil {
			Unauthorized(c, "用户不存在")
			c.Abort()
			return
		}
		if u.Status != models.UserStatusActive {
			Forbidden(c, "用户已禁用")
			c.Abort()
			return
		}
		c.Set(CtxUserID, u.ID)
		c.Set(CtxUsername, u.Username)
		c.Set(CtxRole, u.Role)
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(CtxRole)
		if role != models.RoleAdmin {
			Forbidden(c, "需要管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) uint {
	v, _ := c.Get(CtxUserID)
	id, _ := v.(uint)
	return id
}
