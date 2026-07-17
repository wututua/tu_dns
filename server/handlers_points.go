package server

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func (a *App) handleMyPoints(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	items, total, err := a.points.List(CurrentUserID(c), page, 20)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"items": items, "total": total, "page": page})
}

func (a *App) handleRedeem(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	gained, err := a.redeem.Redeem(CurrentUserID(c), req.Code)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, gin.H{"gained": gained})
}
