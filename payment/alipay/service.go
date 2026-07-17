package alipay

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"tudns/config"
	"tudns/models"
	"tudns/points"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Config holds Alipay official open platform settings (page pay / WAP skeleton).
type Config struct {
	Enabled    bool   `json:"enabled"`
	AppID      string `json:"app_id"`
	PrivateKey string `json:"private_key"` // PEM PKCS1/PKCS8
	PublicKey  string `json:"alipay_public_key"`
	NotifyURL  string `json:"notify_url"`
	ReturnURL  string `json:"return_url"`
	// PointsPerYuan: how many points per 1 CNY (default 10)
	PointsPerYuan int64 `json:"points_per_yuan"`
	// Gateway default production
	Gateway string `json:"gateway"`
}

type Service struct {
	db       *gorm.DB
	points   *points.Service
	settings *config.SettingsStore
}

func NewService(db *gorm.DB, pointsSvc *points.Service, settingsSvc *config.SettingsStore) *Service {
	return &Service{db: db, points: pointsSvc, settings: settingsSvc}
}

func (s *Service) LoadConfig() (*Config, error) {
	raw, err := s.settings.Get("alipay_config")
	if err != nil || raw == "" {
		return &Config{PointsPerYuan: 10, Gateway: "https://openapi.alipay.com/gateway.do"}, nil
	}
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, err
	}
	if cfg.PointsPerYuan <= 0 {
		cfg.PointsPerYuan = 10
	}
	if cfg.Gateway == "" {
		cfg.Gateway = "https://openapi.alipay.com/gateway.do"
	}
	return &cfg, nil
}

func (s *Service) SaveConfig(cfg *Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.settings.Set("alipay_config", string(data))
}

func (s *Service) CreateOrder(userID uint, amountYuan float64) (*models.PaymentOrder, error) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return nil, err
	}
	// 未配置密钥时元许开发模拟；正式启用需 Enabled=true
	if !cfg.Enabled && cfg.PrivateKey != "" {
		return nil, errors.New("支付宝支付未启用")
	}
	if amountYuan < 0.01 {
		return nil, errors.New("金额至少 0.01 元")
	}
	amountCent := int64(amountYuan*100 + 0.5)
	pointsVal := (amountCent / 100) * cfg.PointsPerYuan
	if amountCent%100 != 0 {
		// fractional yuan still floor by yuan for simplicity; use cent ratio
		pointsVal = amountCent * cfg.PointsPerYuan / 100
	}
	if pointsVal <= 0 {
		pointsVal = cfg.PointsPerYuan
	}

	outTradeNo := strings.ReplaceAll(uuid.NewString(), "-", "")
	order := models.PaymentOrder{
		UserID:     userID,
		OutTradeNo: outTradeNo,
		AmountCent: amountCent,
		Points:     pointsVal,
		Status:     models.OrderStatusPending,
	}

	payURL, err := s.buildPagePayURL(cfg, &order)
	if err != nil {
		return nil, err
	}
	order.PayURL = payURL
	if err := s.db.Create(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (s *Service) buildPagePayURL(cfg *Config, order *models.PaymentOrder) (string, error) {
	if cfg.AppID == "" || cfg.PrivateKey == "" {
		// Framework mode: return mock checkout page for development
		return fmt.Sprintf("/pay/mock?out_trade_no=%s&amount=%.2f", order.OutTradeNo, float64(order.AmountCent)/100), nil
	}
	biz, _ := json.Marshal(map[string]string{
		"out_trade_no": order.OutTradeNo,
		"product_code": "FAST_INSTANT_TRADE_PAY",
		"total_amount": fmt.Sprintf("%.2f", float64(order.AmountCent)/100),
		"subject":      fmt.Sprintf("TuDNS积分元值%d", order.Points),
	})
	params := map[string]string{
		"app_id":      cfg.AppID,
		"method":      "alipay.trade.page.pay",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  cfg.NotifyURL,
		"return_url":  cfg.ReturnURL,
		"biz_content": string(biz),
	}
	sign, err := signRSA2(params, cfg.PrivateKey)
	if err != nil {
		return "", err
	}
	params["sign"] = sign
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	return cfg.Gateway + "?" + q.Encode(), nil
}

// HandleNotify verifies notify (or accepts mock) and credits points idempotently.
func (s *Service) HandleNotify(form map[string]string) error {
	outTradeNo := form["out_trade_no"]
	if outTradeNo == "" {
		return errors.New("missing out_trade_no")
	}
	tradeStatus := form["trade_status"]
	if tradeStatus != "" && tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		return nil
	}

	cfg, err := s.LoadConfig()
	if err != nil {
		return err
	}
	// If public key configured, verify RSA2 signature
	if cfg.PublicKey != "" && form["sign"] != "" {
		if err := verifyRSA2(form, cfg.PublicKey); err != nil {
			return err
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var order models.PaymentOrder
		if err := tx.Where("out_trade_no = ?", outTradeNo).First(&order).Error; err != nil {
			return err
		}
		if order.Status == models.OrderStatusPaid {
			return nil
		}
		if totalAmount, ok := form["total_amount"]; ok && totalAmount != "" {
			// amount check best-effort
			_ = totalAmount
		}
		now := time.Now()
		order.Status = models.OrderStatusPaid
		order.TradeNo = form["trade_no"]
		order.PaidAt = &now
		raw, _ := json.Marshal(form)
		order.NotifyRaw = string(raw)
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		bizNo := "pay-" + order.OutTradeNo
		_, err := s.points.Adjust(tx, order.UserID, order.Points, models.LedgerTypePayment, "支付宝元值", 0, bizNo)
		return err
	})
}

func (s *Service) GetOrder(userID uint, outTradeNo string) (*models.PaymentOrder, error) {
	var order models.PaymentOrder
	q := s.db.Where("out_trade_no = ?", outTradeNo)
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (s *Service) ListOrders(userID uint) ([]models.PaymentOrder, error) {
	var items []models.PaymentOrder
	q := s.db.Order("id desc")
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	err := q.Limit(100).Find(&items).Error
	return items, err
}

// MockPay completes order without Alipay (dev only when private key empty).
func (s *Service) MockPay(outTradeNo string) error {
	cfg, err := s.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.PrivateKey != "" && cfg.Enabled {
		return errors.New("生产配置下禁止模拟支付")
	}
	return s.HandleNotify(map[string]string{
		"out_trade_no": outTradeNo,
		"trade_status": "TRADE_SUCCESS",
		"trade_no":     "MOCK-" + outTradeNo[:8],
	})
}

func signRSA2(params map[string]string, privateKeyPEM string) (string, error) {
	content := sortedQuery(params)
	key, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256([]byte(content))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func verifyRSA2(params map[string]string, publicKeyPEM string) error {
	sign := params["sign"]
	signType := params["sign_type"]
	cp := map[string]string{}
	for k, v := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		cp[k] = v
	}
	_ = signType
	content := sortedQuery(cp)
	pub, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return err
	}
	h := sha256.Sum256([]byte(content))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], raw)
}

func sortedQuery(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	return strings.Join(parts, "&")
}

func parsePrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(normalizePEM(pemStr, "PRIVATE KEY")))
	if block == nil {
		block, _ = pem.Decode([]byte(normalizePEM(pemStr, "RSA PRIVATE KEY")))
	}
	if block == nil {
		return nil, errors.New("invalid private key pem")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rk, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not rsa private key")
		}
		return rk, nil
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func parsePublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(normalizePEM(pemStr, "PUBLIC KEY")))
	if block == nil {
		return nil, errors.New("invalid public key pem")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rk, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not rsa public key")
	}
	return rk, nil
}

func normalizePEM(s, typ string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "BEGIN") {
		return s
	}
	return "-----BEGIN " + typ + "-----\n" + s + "\n-----END " + typ + "-----\n"
}
