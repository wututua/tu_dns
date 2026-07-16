package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"tudns/config"
	"tudns/db"
	"tudns/install"
	"tudns/server"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	cfgPath := os.Getenv("TUDNS_CONFIG")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := os.MkdirAll(cfg.App.DataDir, 0o755); err != nil {
		log.Fatalf("data dir: %v", err)
	}

	// if no config.yaml, write example-based default for first run convenience
	if _, err := os.Stat(cfgPath); errors.Is(err, os.ErrNotExist) {
		_ = writeDefaultConfig(cfgPath)
	}

	var gdb *gorm.DB
	if cfg.IsInstalled() {
		gdb, err = install.OpenInstalled(cfg)
		if err != nil {
			log.Fatalf("open database: %v", err)
		}
		db.Set(gdb)
	}

	engine := server.NewRouter(cfg, gdb)
	// allow re-bind after install without restart: store pointer in gin context? handled in install handler via app reassign
	_ = engine

	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("TuDNS listening on %s (installed=%v)", cfg.Addr(), cfg.IsInstalled())
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	if gdb != nil {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	log.Println("TuDNS stopped")
}

func writeDefaultConfig(path string) error {
	content := `app:
  name: tudns
  host: 0.0.0.0
  port: 8080
  mode: dev
  data_dir: data

security:
  secret_key: change-me-before-production-tudns-secret
  token_ttl_hours: 72

cors:
  allow_origins:
    - "*"
`
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// silence unused import if gin not used here
var _ = gin.Version
