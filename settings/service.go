package settings

import (
	"tudns/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Get(key string) (string, error) {
	var item models.Setting
	if err := s.db.Where("key = ?", key).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return item.Value, nil
}

func (s *Service) Set(key, value string) error {
	item := models.Setting{Key: key, Value: value}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&item).Error
}

func (s *Service) GetAll() (map[string]string, error) {
	var items []models.Setting
	if err := s.db.Find(&items).Error; err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, it := range items {
		m[it.Key] = it.Value
	}
	return m, nil
}
