package admin

import (
	"errors"

	"tudns/auth"
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

func (s *Service) ListUsers(page, pageSize int) ([]models.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	if err := s.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.User
	err := s.db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
}

type UpdateUserInput struct {
	Username *string `json:"username"`
	Email    *string `json:"email"`
	Role     *string `json:"role"`
	Status   *int    `json:"status"`
}

func (s *Service) UpdateUser(id uint, in UpdateUserInput) (*models.User, error) {
	var u models.User
	if err := s.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Username != nil {
		updates["username"] = *in.Username
	}
	if in.Email != nil {
		updates["email"] = *in.Email
	}
	if in.Role != nil {
		if *in.Role != models.RoleUser && *in.Role != models.RoleAdmin && *in.Role != models.RolePremium {
			return nil, errors.New("无效角色")
		}
		updates["role"] = *in.Role
	}
	if in.Status != nil {
		if u.Role == models.RoleAdmin && *in.Status != 1 {
			return nil, errors.New("不能禁用管理员用戀")
		}
		updates["status"] = *in.Status
	}
	if len(updates) == 0 {
		return &u, nil
	}
	if err := s.db.Model(&u).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Service) AdjustPoints(adminID, userID uint, delta int64, remark string) (*models.User, error) {
	if delta == 0 {
		return nil, errors.New("调整量不能为0")
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		_, err := s.points.Adjust(tx, userID, delta, models.LedgerTypeAdmin, remark, adminID, "")
		return err
	})
	if err != nil {
		return nil, err
	}
	var u models.User
	if err := s.db.First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Service) ResetPassword(userID uint, newPass string) error {
	if len(newPass) < 6 {
		return errors.New("密码至少6佀")
	}
	hash, err := auth.HashPassword(newPass)
	if err != nil {
		return err
	}
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", hash).Error
}

func (s *Service) ListLogs(page, pageSize int) ([]models.OperationLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	var total int64
	if err := s.db.Model(&models.OperationLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.OperationLog
	err := s.db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func (s *Service) WriteLog(userID, adminID uint, action, targetType, targetID, ip, message string) {
	if s == nil || s.db == nil {
		return
	}
	_ = s.db.Create(&models.OperationLog{
		UserID:     userID,
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IP:         ip,
		Message:    message,
	}).Error
}
