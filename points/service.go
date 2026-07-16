package points

import (
	"errors"
	"fmt"

	"tudns/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInsufficient = errors.New("积分不足")

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Adjust(tx *gorm.DB, userID uint, delta int64, typ, remark string, operator uint, bizNo string) (*models.PointsLedger, error) {
	if tx == nil {
		tx = s.db
	}
	if bizNo == "" {
		bizNo = uuid.NewString()
	}
	var existing models.PointsLedger
	if err := tx.Where("biz_no = ?", bizNo).First(&existing).Error; err == nil {
		return &existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var user models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
		return nil, err
	}
	next := user.Points + delta
	if next < 0 {
		return nil, ErrInsufficient
	}
	if err := tx.Model(&user).Update("points", next).Error; err != nil {
		return nil, err
	}
	ledger := models.PointsLedger{
		UserID:   userID,
		Delta:    delta,
		Balance:  next,
		Type:     typ,
		BizNo:    bizNo,
		Remark:   remark,
		Operator: operator,
	}
	if err := tx.Create(&ledger).Error; err != nil {
		return nil, err
	}
	return &ledger, nil
}

func (s *Service) Charge(tx *gorm.DB, userID uint, cost int64, remark string) (*models.PointsLedger, error) {
	if cost < 0 {
		return nil, fmt.Errorf("invalid cost")
	}
	if cost == 0 {
		return nil, nil
	}
	return s.Adjust(tx, userID, -cost, models.LedgerTypeCharge, remark, 0, uuid.NewString())
}

func (s *Service) List(userID uint, page, pageSize int) ([]models.PointsLedger, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	q := s.db.Model(&models.PointsLedger{})
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.PointsLedger
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
}
