package server

import (
	"tudns/auth"
	"tudns/middleware"
	"tudns/response"

	"github.com/gin-gonic/gin"
)

func (a *App) handleRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	res, err := a.auth.Register(req.Username, req.Password, req.Email)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, res)
}

func (a *App) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	res, err := a.auth.Login(req.Username, req.Password)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, res)
}

func (a *App) handleMe(c *gin.Context) {
	u, err := a.auth.GetUser(middleware.CurrentUserID(c))
	if err != nil {
		response.Unauthorized(c, "用户不存在")
		return
	}
	response.OK(c, auth.ToUserView(u))
}

func (a *App) handleChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	if err := a.auth.ChangePassword(middleware.CurrentUserID(c), req.OldPassword, req.NewPassword); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}
