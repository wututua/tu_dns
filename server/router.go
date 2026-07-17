package server

import (
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"tudns/admin"
	"tudns/auth"
	"tudns/config"
	"tudns/db"
	_ "tudns/dns/providers"
	"tudns/domain"
	"tudns/install"

	"tudns/payment/alipay"
	"tudns/points"
	"tudns/record"
	"tudns/redeem"

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
	settings *config.SettingsStore
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
	settingsSvc := config.NewSettingsStore(gdb)
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

		authg := api.Group("", app.requireInstalled, BearerAuth(cfg))
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

			adminG := authg.Group("/admin", AdminOnly())
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
					NotFound(c, "not found")
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
			NotFound(c, "not found")
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
		Fail(c, http.StatusServiceUnavailable, 503, "系统未安装")
		c.Abort()
		return
	}
	c.Next()
}

func (a *App) handleInstallStatus(c *gin.Context) {
	OK(c, a.install.Status())
}

func (a *App) handleInstallTestDB(c *gin.Context) {
	if a.cfg.IsInstalled() {
		BadRequest(c, "系统已安装")
		return
	}
	var req install.InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	if err := a.install.TestDB(req.Driver, req.DSN, req.SQLitePath); err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, gin.H{"ok": true})
}

func (a *App) handleInstall(c *gin.Context) {
	if a.cfg.IsInstalled() {
		BadRequest(c, "系统已安装")
		return
	}
	var req install.InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误")
		return
	}
	if err := a.install.Install(req); err != nil {
		BadRequest(c, err.Error())
		return
	}
	gdb, err := install.OpenInstalled(a.cfg)
	if err != nil {
		ServerError(c, err.Error())
		return
	}
	a.wire(gdb)
	OK(c, gin.H{"installed": true})
}
