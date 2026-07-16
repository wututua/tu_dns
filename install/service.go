package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"tudns/auth"
	"tudns/config"
	"tudns/db"
	"tudns/models"

	"gorm.io/gorm"
)

type Service struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

type Status struct {
	Installed bool   `json:"installed"`
	AppName   string `json:"app_name"`
}

type InstallRequest struct {
	Driver     string `json:"driver"`
	DSN        string `json:"dsn"`
	SQLitePath string `json:"sqlite_path"`
	AdminUser  string `json:"admin_user"`
	AdminPass  string `json:"admin_pass"`
	AdminEmail string `json:"admin_email"`
	SiteName   string `json:"site_name"`
}

func (s *Service) Status() Status {
	return Status{Installed: s.cfg.IsInstalled(), AppName: s.cfg.App.Name}
}

func (s *Service) TestDB(driver, dsn, sqlitePath string) error {
	driver = strings.ToLower(strings.TrimSpace(driver))
	if driver == "" {
		return errors.New("请选择数据库类垀")
	}
	return db.TestConnection(driver, dsn, sqlitePath, s.cfg.App.DataDir)
}

func (s *Service) Install(req InstallRequest) error {
	if s.cfg.IsInstalled() {
		return errors.New("系统已安裀")
	}
	driver := strings.ToLower(strings.TrimSpace(req.Driver))
	if driver == "" {
		return errors.New("请选择数据库类垀")
	}
	if len(req.AdminUser) < 3 || len(req.AdminPass) < 6 {
		return errors.New("管理员用户名至少3位，密码至少6佀")
	}
	if err := os.MkdirAll(s.cfg.App.DataDir, 0o755); err != nil {
		return err
	}

	dbCfg := config.DatabaseConfig{
		Driver: driver,
		DSN:    strings.TrimSpace(req.DSN),
		Path:   strings.TrimSpace(req.SQLitePath),
	}
	if driver == "sqlite" {
		if dbCfg.Path == "" {
			dbCfg.Path = "tudns.db"
		}
	}

	s.cfg.Database = dbCfg
	gdb, err := db.Open(s.cfg)
	if err != nil {
		return err
	}

	// ensure no existing admin when fresh? allow overwrite only if empty users
	var count int64
	if err := gdb.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("数据库中已有用户，请使用空库安装")
	}

	authSvc := auth.NewService(gdb, s.cfg)
	if _, err := authSvc.CreateAdmin(req.AdminUser, req.AdminPass, req.AdminEmail); err != nil {
		return err
	}

	if req.SiteName != "" {
		_ = gdb.Create(&models.Setting{Key: "site_name", Value: req.SiteName}).Error
	} else {
		_ = gdb.Create(&models.Setting{Key: "site_name", Value: "TuDNS"}).Error
	}

	if err := config.SaveDatabaseConfig(s.cfg, dbCfg); err != nil {
		return err
	}
	if err := config.WriteInstallLock(s.cfg); err != nil {
		return err
	}
	// ensure absolute sqlite path recorded
	_ = filepath.Join(s.cfg.App.DataDir, dbCfg.Path)
	return nil
}

func OpenInstalled(cfg *config.Config) (*gorm.DB, error) {
	if !cfg.IsInstalled() {
		return nil, errors.New("not installed")
	}
	if err := config.LoadDatabaseConfig(cfg); err != nil {
		return nil, err
	}
	return db.Open(cfg)
}
