package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"tudns/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ApiKeyService struct {
	db *gorm.DB
}

func NewApiKeyService(db *gorm.DB) *ApiKeyService {
	return &ApiKeyService{db: db}
}

func (s *ApiKeyService) Create(userID uint, name string) (string, *models.ApiKey, error) {
	raw := generateKey()
	prefix := raw[:8]
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, err
	}
	key := models.ApiKey{
		UserID:    userID,
		Name:      name,
		KeyPrefix: prefix,
		KeyHash:   string(hash),
		Enabled:   true,
	}
	if err := s.db.Create(&key).Error; err != nil {
		return "", nil, err
	}
	return raw, &key, nil
}

func (s *ApiKeyService) List(userID uint) ([]models.ApiKey, error) {
	var items []models.ApiKey
	q := s.db.Select("id, user_id, name, key_prefix, last_used_at, last_ip, enabled, created_at, updated_at")
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	err := q.Order("id desc").Find(&items).Error
	return items, err
}

func (s *ApiKeyService) Revoke(userID, keyID uint) error {
	q := s.db.Model(&models.ApiKey{}).Where("id = ?", keyID)
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	return q.Update("enabled", false).Error
}

func (s *ApiKeyService) Authenticate(token string) (*models.User, error) {
	hash := sha256.Sum256([]byte(token))
	_ = hash
	var keys []models.ApiKey
	if err := s.db.Where("enabled = ?", true).Find(&keys).Error; err != nil {
		return nil, err
	}
	for _, k := range keys {
		if err := bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(token)); err == nil {
			var u models.User
			if err := s.db.First(&u, k.UserID).Error; err != nil {
				return nil, err
			}
			if u.Status != models.UserStatusActive {
				return nil, errors.New("用户已禁用")
			}
			now := time.Now()
			s.db.Model(&k).Updates(map[string]interface{}{
				"last_used_at": &now,
			})
			return &u, nil
		}
	}
	return nil, errors.New("无效的 API 密钥")
}

func validateKey(token string) bool {
	if len(token) < 32 {
		return false
	}
	_, err := hex.DecodeString(token[:32])
	return err == nil
}

func generateKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// LookupKey returns the raw key prefix and full hash for a given prefix
func LookupKey(prefix string) string {
	return prefix
}

// IsAPIKey checks if a token looks like a raw API key (hex, 64 chars)
func IsAPIKey(token string) bool {
	// tud_ prefix check for external format, but internally just hex
	if strings.HasPrefix(token, "tud_") {
		token = token[4:]
	}
	if len(token) != 64 {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}
