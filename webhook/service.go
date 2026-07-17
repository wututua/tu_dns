package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"tudns/models"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List() ([]models.Webhook, error) {
	var items []models.Webhook
	err := s.db.Order("id desc").Find(&items).Error
	return items, err
}

func (s *Service) Create(name, url, events, secret string) (*models.Webhook, error) {
	w := models.Webhook{
		Name:    name,
		URL:     url,
		Events:  events,
		Secret:  secret,
		Enabled: true,
	}
	if err := s.db.Create(&w).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *Service) Update(id uint, name, url, events, secret string, enabled bool) (*models.Webhook, error) {
	var w models.Webhook
	if err := s.db.First(&w, id).Error; err != nil {
		return nil, err
	}
	w.Name = name
	w.URL = url
	w.Events = events
	if secret != "" {
		w.Secret = secret
	}
	w.Enabled = enabled
	if err := s.db.Save(&w).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *Service) Delete(id uint) error {
	return s.db.Delete(&models.Webhook{}, id).Error
}

type EventPayload struct {
	Event     string      `json:"event"`
	CreatedAt time.Time   `json:"created_at"`
	Data      interface{} `json:"data"`
}

func (s *Service) Dispatch(event string, data interface{}) {
	var hooks []models.Webhook
	if err := s.db.Where("enabled = ?", true).Find(&hooks).Error; err != nil {
		return
	}
	payload := EventPayload{
		Event:     event,
		CreatedAt: time.Now(),
		Data:      data,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	for _, h := range hooks {
		if !s.matchesEvent(h.Events, event) {
			continue
		}
		go s.send(h, body)
	}
}

func (s *Service) matchesEvent(events, event string) bool {
	for _, e := range strings.Split(events, ",") {
		if strings.TrimSpace(e) == "*" || strings.TrimSpace(e) == event {
			return true
		}
	}
	return false
}

func (s *Service) send(h models.Webhook, body []byte) {
	req, err := http.NewRequest(http.MethodPost, h.URL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", "")
	if h.Secret != "" {
		mac := hmac.New(sha256.New, []byte(h.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Webhook-Signature", fmt.Sprintf("sha256=%s", sig))
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("webhook %s: %v", h.URL, err)
		return
	}
	_ = resp.Body.Close()
}
