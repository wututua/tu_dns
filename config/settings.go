package config

import (
	"tudns/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingsStore struct {
	db *gorm.DB
}

func NewSettingsStore(db *gorm.DB) *SettingsStore {
	return &SettingsStore{db: db}
}

func (s *SettingsStore) Get(key string) (string, error) {
	var item models.Setting
	if err := s.db.Where("key = ?", key).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return item.Value, nil
}

func (s *SettingsStore) Set(key, value string) error {
	item := models.Setting{Key: key, Value: value}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&item).Error
}

func (s *SettingsStore) GetAll() (map[string]string, error) {
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
