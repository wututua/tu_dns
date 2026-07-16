package auth

import (
	"errors"
	"strings"
	"time"

	"tudns/config"
	"tudns/models"

	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserDisabled       = errors.New("用户已禁用")
	ErrUserExists         = errors.New("用户名已存在")
)

type Service struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

type TokenResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserView  `json:"user"`
}

type UserView struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    int       `json:"status"`
	Points    int64     `json:"points"`
	CreatedAt time.Time `json:"created_at"`
}

func ToUserView(u *models.User) UserView {
	return UserView{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Status:    u.Status,
		Points:    u.Points,
		CreatedAt: u.CreatedAt,
	}
}

func (s *Service) Register(username, password, email string) (*TokenResult, error) {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(password) < 6 {
		return nil, errors.New("用户名至少位，密码至少6位")
	}
	var count int64
	if err := s.db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrUserExists
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := models.User{
		Username:     username,
		Email:        strings.TrimSpace(email),
		PasswordHash: hash,
		Role:         models.RoleUser,
		Status:       models.UserStatusActive,
		Points:       0,
	}
	if err := s.db.Create(&u).Error; err != nil {
		return nil, err
	}
	return s.issue(&u)
}

func (s *Service) Login(username, password string) (*TokenResult, error) {
	var u models.User
	if err := s.db.Where("username = ?", strings.TrimSpace(username)).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if u.Status != models.UserStatusActive {
		return nil, ErrUserDisabled
	}
	if !CheckPassword(u.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	return s.issue(&u)
}

func (s *Service) issue(u *models.User) (*TokenResult, error) {
	token, exp, err := IssueToken(s.cfg.Security.SecretKey, s.cfg.Security.TokenTTLHours, u.ID, u.Username, u.Role)
	if err != nil {
		return nil, err
	}
	return &TokenResult{Token: token, ExpiresAt: exp, User: ToUserView(u)}, nil
}

func (s *Service) GetUser(id uint) (*models.User, error) {
	var u models.User
	if err := s.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Service) ChangePassword(userID uint, oldPwd, newPwd string) error {
	if len(newPwd) < 6 {
		return errors.New("新密码至少位")
	}
	var u models.User
	if err := s.db.First(&u, userID).Error; err != nil {
		return err
	}
	if !CheckPassword(u.PasswordHash, oldPwd) {
		return errors.New("原密码错误")
	}
	hash, err := HashPassword(newPwd)
	if err != nil {
		return err
	}
	return s.db.Model(&u).Update("password_hash", hash).Error
}

func (s *Service) CreateAdmin(username, password, email string) (*models.User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := models.User{
		Username:     strings.TrimSpace(username),
		Email:        strings.TrimSpace(email),
		PasswordHash: hash,
		Role:         models.RoleAdmin,
		Status:       models.UserStatusActive,
		Points:       0,
	}
	if err := s.db.Create(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
