package server

import (
	"strconv"
	"time"

	"tudns/admin"
	"tudns/auth"
	"tudns/domain"

	"tudns/payment/alipay"


	"github.com/gin-gonic/gin"
)

func (a *App) handleAdminUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	items, total, err := a.admin.ListUsers(page, 20)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	views := make([]auth.UserView, 0, len(items))
	for i := range items {
		views = append(views, auth.ToUserView(&items[i]))
	}
	OK(c, gin.H{"items": views, "total": total, "page": page})
}

func (a *App) handleAdminUpdateUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req admin.UpdateUserInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	u, err := a.admin.UpdateUser(uint(id), req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, auth.ToUserView(u))
}

func (a *App) handleAdminAdjustPoints(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Delta  int64  `json:"delta"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	u, err := a.admin.AdjustPoints(CurrentUserID(c), uint(id), req.Delta, req.Remark)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, auth.ToUserView(u))
}

func (a *App) handleAdminResetPassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	if err := a.admin.ResetPassword(uint(id), req.Password); err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}

func (a *App) handleAdminDomains(c *gin.Context) {
	items, err := a.domain.ListAll()
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, items)
}

func (a *App) handleAdminSaveDomain(c *gin.Context) {
	var id uint
	if c.Param("id") != "" {
		v, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		id = uint(v)
	}
	var req domain.SaveInput
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	d, err := a.domain.Save(id, req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, d)
}

func (a *App) handleAdminDeleteDomain(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := a.domain.Delete(uint(id)); err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}

func (a *App) handleAdminDNSCheck(c *gin.Context) {
	var req struct {
		ProviderKey string            `json:"provider_key"`
		Config      map[string]string `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	if err := a.domain.CheckProvider(c.Request.Context(), req.ProviderKey, req.Config); err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}

func (a *App) handleAdminDNSZones(c *gin.Context) {
	var req struct {
		ProviderKey string            `json:"provider_key"`
		Config      map[string]string `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	zones, err := a.domain.ListZones(c.Request.Context(), req.ProviderKey, req.Config)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, zones)
}

func (a *App) handleAdminSubdomains(c *gin.Context) {
	items, err := a.record.ListSubdomains(0)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, items)
}

func (a *App) handleAdminRecords(c *gin.Context) {
	items, err := a.record.ListAll()
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, items)
}

func (a *App) handleAdminPoints(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	items, total, err := a.points.List(0, page, 20)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"items": items, "total": total, "page": page})
}

func (a *App) handleAdminLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	items, total, err := a.admin.ListLogs(page, 20)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"items": items, "total": total, "page": page})
}

func (a *App) handleAdminListRedeem(c *gin.Context) {
	items, err := a.redeem.List()
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, items)
}

func (a *App) handleAdminCreateRedeem(c *gin.Context) {
	var req struct {
		Points    int64  `json:"points"`
		MaxUses   int    `json:"max_uses"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	var exp *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			BadRequest(c, "过期时间格式错误")
			return
		}
		exp = &t
	}
	item, err := a.redeem.Create(req.Points, req.MaxUses, exp)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, item)
}

func (a *App) handleAdminGetSettings(c *gin.Context) {
	m, err := a.settings.GetAll()
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, m)
}

func (a *App) handleAdminSaveSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	for k, v := range req {
		if err := a.settings.Set(k, v); err != nil {
			ServerError(c, err.Error())
			return
		}
	}
	OK(c, gin.H{"ok": true})
}

func (a *App) handleAdminGetAlipay(c *gin.Context) {
	cfg, err := a.alipay.LoadConfig()
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	if cfg.PrivateKey != "" {
		cfg.PrivateKey = "***"
	}
	if cfg.PublicKey != "" {
		cfg.PublicKey = "***"
	}
	OK(c, cfg)
}

func (a *App) handleAdminSaveAlipay(c *gin.Context) {
	var cfg alipay.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	old, _ := a.alipay.LoadConfig()
	if cfg.PrivateKey == "***" || cfg.PrivateKey == "" {
		cfg.PrivateKey = old.PrivateKey
	}
	if cfg.PublicKey == "***" || cfg.PublicKey == "" {
		cfg.PublicKey = old.PublicKey
	}
	if err := a.alipay.SaveConfig(&cfg); err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}

func (a *App) handleAdminOrders(c *gin.Context) {
	items, err := a.alipay.ListOrders(0)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	OK(c, items)
}
