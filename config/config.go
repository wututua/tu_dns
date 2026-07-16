package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	Security SecurityConfig `yaml:"security"`
	CORS     CORSConfig     `yaml:"cors"`
	Database DatabaseConfig `yaml:"database"`
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Mode    string `yaml:"mode"`
	DataDir string `yaml:"data_dir"`
}

type SecurityConfig struct {
	SecretKey     string `yaml:"secret_key"`
	TokenTTLHours int    `yaml:"token_ttl_hours"`
	InstallToken  string `yaml:"install_token"`
}

type CORSConfig struct {
	AllowOrigins []string `yaml:"allow_origins"`
}

// DatabaseConfig is written during install wizard.
type DatabaseConfig struct {
	Driver string `yaml:"driver"` // sqlite | mysql | postgres
	DSN    string `yaml:"dsn"`
	Path   string `yaml:"path"` // sqlite relative path under data_dir
}

var (
	mu     sync.RWMutex
	global *Config
)

func Default() *Config {
	return &Config{
		App: AppConfig{
			Name:    "tudns",
			Host:    "0.0.0.0",
			Port:    8080,
			Mode:    "release",
			DataDir: "data",
		},
		Security: SecurityConfig{
			SecretKey:     "change-me-before-production-tudns-secret",
			TokenTTLHours: 72,
		},
		CORS: CORSConfig{AllowOrigins: []string{"*"}},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			global = cfg
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.App.Port == 0 {
		cfg.App.Port = 8080
	}
	if cfg.App.DataDir == "" {
		cfg.App.DataDir = "data"
	}
	if cfg.Security.TokenTTLHours <= 0 {
		cfg.Security.TokenTTLHours = 72
	}
	global = cfg
	return cfg, nil
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	if global == nil {
		return Default()
	}
	return global
}

func Set(cfg *Config) {
	mu.Lock()
	defer mu.Unlock()
	global = cfg
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.App.Host, c.App.Port)
}

func (c *Config) InstallLockPath() string {
	return filepath.Join(c.App.DataDir, "install.lock")
}

func (c *Config) DatabaseConfigPath() string {
	return filepath.Join(c.App.DataDir, "database.yaml")
}

func (c *Config) IsInstalled() bool {
	_, err := os.Stat(c.InstallLockPath())
	return err == nil
}

func SaveDatabaseConfig(cfg *Config, db DatabaseConfig) error {
	if err := os.MkdirAll(cfg.App.DataDir, 0o755); err != nil {
		return err
	}
	cfg.Database = db
	data, err := yaml.Marshal(struct {
		Database DatabaseConfig `yaml:"database"`
	}{Database: db})
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.DatabaseConfigPath(), data, 0o600)
}

func LoadDatabaseConfig(cfg *Config) error {
	path := cfg.DatabaseConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var wrapper struct {
		Database DatabaseConfig `yaml:"database"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return err
	}
	cfg.Database = wrapper.Database
	return nil
}

func WriteInstallLock(cfg *Config) error {
	if err := os.MkdirAll(cfg.App.DataDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(cfg.InstallLockPath(), []byte("installed\n"), 0o600)
}
