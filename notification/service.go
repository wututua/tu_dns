package notification

import (
	"tudns/models"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(userID uint, title, content, link string) error {
	n := models.Notification{
		UserID:  userID,
		Title:   title,
		Content: content,
		Link:    link,
	}
	return s.db.Create(&n).Error
}

func (s *Service) List(userID uint, page, pageSize int) ([]models.Notification, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}
	q := s.db.Model(&models.Notification{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.Notification
	err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func (s *Service) MarkRead(userID, id uint) error {
	return s.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("read", true).Error
}

func (s *Service) MarkAllRead(userID uint) error {
	return s.db.Model(&models.Notification{}).
		Where("user_id = ? AND `read` = ?", userID, false).
		Update("read", true).Error
}

func (s *Service) UnreadCount(userID uint) (int64, error) {
	var count int64
	err := s.db.Model(&models.Notification{}).
		Where("user_id = ? AND `read` = ?", userID, false).
		Count(&count).Error
	return count, err
}
