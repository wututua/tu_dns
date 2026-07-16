package server

import (
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"tudns/admin"
	"tudns/auth"
	"tudns/config"
	"tudns/db"
	"tudns/dns"
	_ "tudns/dns/providers"
	"tudns/domain"
	"tudns/install"
	"tudns/middleware"
	"tudns/payment/alipay"
	"tudns/points"
	"tudns/record"
	"tudns/redeem"
	"tudns/response"
	"tudns/settings"
	"tudns/webembed"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type App struct {
	mu       sync.RWMutex
	cfg      *config.Config
	db       *gorm.DB
	auth     *auth.Service
	install  *install.Service
	domain   *domain.Service
	record   *record.Service
	points   *points.Service
	redeem   *redeem.Service
	alipay   *alipay.Service
	admin    *admin.Service
	settings *settings.Service
}

func NewApp(cfg *config.Config, gdb *gorm.DB) *App {
	a := &App{cfg: cfg, install: install.NewService(cfg)}
	if gdb != nil {
		a.wire(gdb)
	}
	return a
}

func (a *App) wire(gdb *gorm.DB) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.db = gdb
	db.Set(gdb)
	pointsSvc := points.NewService(gdb)
	settingsSvc := settings.NewService(gdb)
	domainSvc := domain.NewService(gdb, a.cfg)
	a.auth = auth.NewService(gdb, a.cfg)
	a.domain = domainSvc
	a.record = record.NewService(gdb, domainSvc, pointsSvc)
	a.points = pointsSvc
	a.redeem = redeem.NewService(gdb, pointsSvc)
	a.alipay = alipay.NewService(gdb, pointsSvc, settingsSvc)
	a.admin = admin.NewService(gdb, pointsSvc)
	a.settings = settingsSvc
}

func NewRouter(cfg *config.Config, gdb *gorm.DB) *gin.Engine {
	if cfg.App.Mode == "dev" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), corsMiddleware(cfg), securityHeaders())

	app := NewApp(cfg, gdb)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		if !cfg.IsInstalled() {
			c.JSON(http.StatusOK, gin.H{"ready": false, "reason": "not_installed"})
			return
		}
		current := db.Get()
		if current == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false, "reason": "no_db"})
			return
		}
		sqlDB, err := current.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false, "reason": "db"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ready": true})
	})

	api := r.Group("/api")
	{
		api.GET("/install/status", app.handleInstallStatus)
		api.POST("/install/test-db", app.handleInstallTestDB)
		api.POST("/install", app.handleInstall)

		api.POST("/auth/register", app.requireInstalled, app.handleRegister)
		api.POST("/auth/login", app.requireInstalled, app.handleLogin)

		api.GET("/public/domains", app.requireInstalled, app.handlePublicDomains)
		api.GET("/dns/providers", app.requireInstalled, app.handleDNSProviders)

		api.POST("/pay/alipay/notify", app.requireInstalled, app.handleAlipayNotify)
		api.POST("/pay/alipay/mock", app.requireInstalled, app.handleAlipayMock)

		authg := api.Group("", app.requireInstalled, middleware.BearerAuth(cfg))
		{
			authg.GET("/auth/me", app.handleMe)
			authg.PUT("/auth/password", app.handleChangePassword)

			authg.GET("/subdomains", app.handleMySubdomains)
			authg.POST("/subdomains/bundle", app.handleCreateBundle)
			authg.DELETE("/subdomains/:id", app.handleDeleteSubdomain)

			authg.GET("/records", app.handleMyRecords)
			authg.POST("/records", app.handleAddRecord)
			authg.PUT("/records/:id", app.handleUpdateRecord)
			authg.DELETE("/records/:id", app.handleDeleteRecord)

			authg.GET("/points", app.handleMyPoints)
			authg.POST("/redeem", app.handleRedeem)

			authg.POST("/pay/alipay/create", app.handleAlipayCreate)
			authg.GET("/pay/orders", app.handleMyOrders)
			authg.GET("/pay/orders/:out_trade_no", app.handleGetOrder)

			adminG := authg.Group("/admin", middleware.AdminOnly())
			{
				adminG.GET("/users", app.handleAdminUsers)
				adminG.PUT("/users/:id", app.handleAdminUpdateUser)
				adminG.POST("/users/:id/points", app.handleAdminAdjustPoints)
				adminG.POST("/users/:id/password", app.handleAdminResetPassword)

				adminG.GET("/domains", app.handleAdminDomains)
				adminG.POST("/domains", app.handleAdminSaveDomain)
				adminG.PUT("/domains/:id", app.handleAdminSaveDomain)
				adminG.DELETE("/domains/:id", app.handleAdminDeleteDomain)
				adminG.POST("/dns/check", app.handleAdminDNSCheck)
				adminG.POST("/dns/zones", app.handleAdminDNSZones)

				adminG.GET("/subdomains", app.handleAdminSubdomains)
				adminG.GET("/records", app.handleAdminRecords)
				adminG.GET("/points", app.handleAdminPoints)
				adminG.GET("/logs", app.handleAdminLogs)

				adminG.GET("/redeem", app.handleAdminListRedeem)
				adminG.POST("/redeem", app.handleAdminCreateRedeem)

				adminG.GET("/settings", app.handleAdminGetSettings)
				adminG.PUT("/settings", app.handleAdminSaveSettings)
				adminG.GET("/pay/alipay/config", app.handleAdminGetAlipay)
				adminG.PUT("/pay/alipay/config", app.handleAdminSaveAlipay)
				adminG.GET("/pay/orders", app.handleAdminOrders)
			}
		}
	}

	staticFS, err := webembed.FS()
	if err == nil {
		sub, err := fs.Sub(staticFS, "dist")
		if err == nil {
			fileServer := http.FileServer(http.FS(sub))
			r.NoRoute(func(c *gin.Context) {
				if strings.HasPrefix(c.Request.URL.Path, "/api") {
					response.NotFound(c, "not found")
					return
				}
				path := strings.TrimPrefix(c.Request.URL.Path, "/")
				if path != "" {
					if f, err := sub.Open(path); err == nil {
						_ = f.Close()
						fileServer.ServeHTTP(c.Writer, c.Request)
						return
					}
				}
				data, err := fs.ReadFile(sub, "index.html")
				if err != nil {
					c.String(http.StatusOK, "TuDNS API is running. Run the frontend production build.")
					return
				}
				c.Data(http.StatusOK, "text/html; charset=utf-8", data)
			})
			return r
		}
	}
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			response.NotFound(c, "not found")
			return
		}
		c.String(http.StatusOK, "TuDNS API is running. Run the frontend production build.")
	})
	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := "*"
		if len(cfg.CORS.AllowOrigins) > 0 {
			origin = cfg.CORS.AllowOrigins[0]
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Next()
	}
}

func (a *App) requireInstalled(c *gin.Context) {
	if !a.cfg.IsInstalled() || db.Get() == nil {
		response.Fail(c, http.StatusServiceUnavailable, 503, "系统未安裀")
		c.Abort()
		return
	}
	c.Next()
}

func (a *App) handleInstallStatus(c *gin.Context) {
	response.OK(c, a.install.Status())
}

func (a *App) handleInstallTestDB(c *gin.Context) {
	if a.cfg.IsInstalled() {
		response.BadRequest(c, "系统已安裀")
		return
	}
	var req install.InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	if err := a.install.TestDB(req.Driver, req.DSN, req.SQLitePath); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) handleInstall(c *gin.Context) {
	if a.cfg.IsInstalled() {
		response.BadRequest(c, "系统已安裀")
		return
	}
	var req install.InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	if err := a.install.Install(req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	gdb, err := install.OpenInstalled(a.cfg)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	a.wire(gdb)
	response.OK(c, gin.H{"installed": true})
}

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
		response.Unauthorized(c, "用户不存圀")
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

func (a *App) handlePublicDomains(c *gin.Context) {
	items, err := a.domain.ListPublic()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (a *App) handleDNSProviders(c *gin.Context) {
	response.OK(c, dns.List())
}

func (a *App) handleCreateBundle(c *gin.Context) {
	var req record.BundleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	uid := middleware.CurrentUserID(c)
	res, err := a.record.CreateBundle(uid, req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	a.admin.WriteLog(uid, 0, "create_bundle", "subdomain", strconv.FormatUint(uint64(res.Subdomain.ID), 10), c.ClientIP(), res.Subdomain.FullDomain)
	response.OK(c, res)
}

func (a *App) handleMySubdomains(c *gin.Context) {
	items, err := a.record.ListSubdomains(middleware.CurrentUserID(c))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (a *App) handleDeleteSubdomain(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	uid := middleware.CurrentUserID(c)
	role, _ := c.Get(middleware.CtxRole)
	if err := a.record.DeleteSubdomain(uid, uint(id), role == "admin"); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) handleMyRecords(c *gin.Context) {
	items, err := a.record.ListByUser(middleware.CurrentUserID(c))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (a *App) handleAddRecord(c *gin.Context) {
	var req record.AddRecordInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	uid := middleware.CurrentUserID(c)
	role, _ := c.Get(middleware.CtxRole)
	rec, charged, err := a.record.AddRecord(uid, req, role == "admin")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"record": rec, "charged": charged})
}

func (a *App) handleUpdateRecord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req record.UpdateRecordInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	uid := middleware.CurrentUserID(c)
	role, _ := c.Get(middleware.CtxRole)
	rec, err := a.record.UpdateRecord(uid, uint(id), req, role == "admin")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, rec)
}

func (a *App) handleDeleteRecord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	uid := middleware.CurrentUserID(c)
	role, _ := c.Get(middleware.CtxRole)
	if err := a.record.DeleteRecord(uid, uint(id), role == "admin"); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) handleMyPoints(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	items, total, err := a.points.List(middleware.CurrentUserID(c), page, 20)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items, "total": total, "page": page})
}

func (a *App) handleRedeem(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	gained, err := a.redeem.Redeem(middleware.CurrentUserID(c), req.Code)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"gained": gained})
}

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
		response.NotFound(c, "订单不存圀")
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

func (a *App) handleAdminUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	items, total, err := a.admin.ListUsers(page, 20)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	views := make([]auth.UserView, 0, len(items))
	for i := range items {
		views = append(views, auth.ToUserView(&items[i]))
	}
	response.OK(c, gin.H{"items": views, "total": total, "page": page})
}

func (a *App) handleAdminUpdateUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req admin.UpdateUserInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	u, err := a.admin.UpdateUser(uint(id), req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, auth.ToUserView(u))
}

func (a *App) handleAdminAdjustPoints(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Delta  int64  `json:"delta"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	u, err := a.admin.AdjustPoints(middleware.CurrentUserID(c), uint(id), req.Delta, req.Remark)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, auth.ToUserView(u))
}

func (a *App) handleAdminResetPassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	if err := a.admin.ResetPassword(uint(id), req.Password); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) handleAdminDomains(c *gin.Context) {
	items, err := a.domain.ListAll()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (a *App) handleAdminSaveDomain(c *gin.Context) {
	var id uint
	if c.Param("id") != "" {
		v, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		id = uint(v)
	}
	var req domain.SaveInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	d, err := a.domain.Save(id, req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, d)
}

func (a *App) handleAdminDeleteDomain(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := a.domain.Delete(uint(id)); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) handleAdminDNSCheck(c *gin.Context) {
	var req struct {
		ProviderKey string            `json:"provider_key"`
		Config      map[string]string `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	if err := a.domain.CheckProvider(c.Request.Context(), req.ProviderKey, req.Config); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) handleAdminDNSZones(c *gin.Context) {
	var req struct {
		ProviderKey string            `json:"provider_key"`
		Config      map[string]string `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	zones, err := a.domain.ListZones(c.Request.Context(), req.ProviderKey, req.Config)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, zones)
}

func (a *App) handleAdminSubdomains(c *gin.Context) {
	items, err := a.record.ListSubdomains(0)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (a *App) handleAdminRecords(c *gin.Context) {
	items, err := a.record.ListAll()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (a *App) handleAdminPoints(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	items, total, err := a.points.List(0, page, 20)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items, "total": total, "page": page})
}

func (a *App) handleAdminLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	items, total, err := a.admin.ListLogs(page, 20)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"items": items, "total": total, "page": page})
}

func (a *App) handleAdminListRedeem(c *gin.Context) {
	items, err := a.redeem.List()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}

func (a *App) handleAdminCreateRedeem(c *gin.Context) {
	var req struct {
		Points    int64  `json:"points"`
		MaxUses   int    `json:"max_uses"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	var exp *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			response.BadRequest(c, "过期时间格式错误")
			return
		}
		exp = &t
	}
	item, err := a.redeem.Create(req.Points, req.MaxUses, exp)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, item)
}

func (a *App) handleAdminGetSettings(c *gin.Context) {
	m, err := a.settings.GetAll()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, m)
}

func (a *App) handleAdminSaveSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	for k, v := range req {
		if err := a.settings.Set(k, v); err != nil {
			response.ServerError(c, err.Error())
			return
		}
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) handleAdminGetAlipay(c *gin.Context) {
	cfg, err := a.alipay.LoadConfig()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	if cfg.PrivateKey != "" {
		cfg.PrivateKey = "***"
	}
	if cfg.PublicKey != "" {
		cfg.PublicKey = "***"
	}
	response.OK(c, cfg)
}

func (a *App) handleAdminSaveAlipay(c *gin.Context) {
	var cfg alipay.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, "参数错误")
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
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func (a *App) handleAdminOrders(c *gin.Context) {
	items, err := a.alipay.ListOrders(0)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, items)
}
