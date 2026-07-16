package redeem

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"tudns/models"
	"tudns/points"

	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	points *points.Service
}

func NewService(db *gorm.DB, pointsSvc *points.Service) *Service {
	return &Service{db: db, points: pointsSvc}
}

func (s *Service) Create(pointsVal int64, maxUses int, expiresAt *time.Time) (*models.RedeemCode, error) {
	if pointsVal <= 0 {
		return nil, errors.New("积分必须大于0")
	}
	if maxUses <= 0 {
		maxUses = 1
	}
	code, err := randomCode(12)
	if err != nil {
		return nil, err
	}
	item := models.RedeemCode{
		Code:      code,
		Points:    pointsVal,
		MaxUses:   maxUses,
		ExpiresAt: expiresAt,
		Enabled:   true,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) List() ([]models.RedeemCode, error) {
	var items []models.RedeemCode
	err := s.db.Order("id desc").Find(&items).Error
	return items, err
}

func (s *Service) Redeem(userID uint, code string) (int64, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return 0, errors.New("兑换码不能为空")
	}
	var gained int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var item models.RedeemCode
		if err := tx.Where("code = ?", code).First(&item).Error; err != nil {
			return errors.New("兑换码无效")
		}
		if !item.Enabled {
			return errors.New("兑换码已停用")
		}
		if item.ExpiresAt != nil && item.ExpiresAt.Before(time.Now()) {
			return errors.New("兑换码已过期")
		}
		if item.UsedCount >= item.MaxUses {
			return errors.New("兑换码已用尽")
		}
		var used int64
		if err := tx.Model(&models.RedeemUse{}).Where("code_id = ? AND user_id = ?", item.ID, userID).Count(&used).Error; err != nil {
			return err
		}
		if used > 0 {
			return errors.New("您已兑换过该码")
		}
		bizNo := "redeem-" + item.Code + "-" + time.Now().Format("20060102150405")
		if _, err := s.points.Adjust(tx, userID, item.Points, models.LedgerTypeRedeem, "兑换码"+item.Code, 0, bizNo); err != nil {
			return err
		}
		if err := tx.Model(&item).Updates(map[string]interface{}{
			"used_count": item.UsedCount + 1,
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.RedeemUse{CodeID: item.ID, UserID: userID}).Error; err != nil {
			return err
		}
		gained = item.Points
		return nil
	})
	return gained, err
}

func randomCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(b)[:n]), nil
}
