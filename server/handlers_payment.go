package server

import (
	"net/http"

	"tudns/middleware"
	"tudns/response"

	"github.com/gin-gonic/gin"
)

func (a *App) handleAlipayCreate(c *gin.Context) {
	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	order, err := a.alipay.CreateOrder(middleware.CurrentUserID(c), req.Amount)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, order)
}

func (a *App) handleMyOrders(c *gin.Context) {
	items, err := a.alipay.ListOrders(middleware.CurrentUserID(c))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (a *App) handleGetOrder(c *gin.Context) {
	order, err := a.alipay.GetOrder(middleware.CurrentUserID(c), c.Param("out_trade_no"))
	if err != nil {
		response.NotFound(c, "订单不存在")
		return
	}
	response.OK(c, order)
}

func (a *App) handleAlipayNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	form := map[string]string{}
	for k, v := range c.Request.PostForm {
		if len(v) > 0 {
			form[k] = v[0]
		}
	}
	if err := a.alipay.HandleNotify(form); err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	c.String(http.StatusOK, "success")
}

func (a *App) handleAlipayMock(c *gin.Context) {
	var req struct {
		OutTradeNo string `json:"out_trade_no"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	if err := a.alipay.MockPay(req.OutTradeNo); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}
