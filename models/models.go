package models

import "time"

const (
	RoleUser    = "user"
	RoleAdmin   = "admin"
	RolePremium = "premium"

	UserStatusActive   = 1
	UserStatusDisabled = 0

	DomainStatusActive   = 1
	DomainStatusDisabled = 0

	SubdomainStatusActive   = 1
	SubdomainStatusDisabled = 0

	LedgerTypeAdmin   = "admin"
	LedgerTypeRedeem  = "redeem"
	LedgerTypePayment = "payment"
	LedgerTypeCharge  = "charge"
	LedgerTypeRefund  = "refund"

	OrderStatusPending = "pending"
	OrderStatusPaid    = "paid"
	OrderStatusClosed  = "closed"
	OrderStatusFailed  = "failed"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Email        string    `gorm:"size:128;index" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:16;not null;default:user" json:"role"`
	Status       int       `gorm:"not null;default:1" json:"status"`
	Points       int64     `gorm:"not null;default:0" json:"points"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Domain struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Name             string    `gorm:"size:255;uniqueIndex;not null" json:"name"`
	ProviderKey      string    `gorm:"size:32;not null" json:"provider_key"`
	RemoteZoneID     string    `gorm:"size:128" json:"remote_zone_id"`
	ConfigCiphertext string    `gorm:"type:text" json:"-"`
	RecordTypes      string    `gorm:"size:255;not null;default:A,AAAA,CNAME,TXT" json:"record_types"`
	PointsCost       int64     `gorm:"not null;default:0" json:"points_cost"`
	Description      string    `gorm:"size:512" json:"description"`
	SubdomainTTLDays int       `gorm:"not null;default:0" json:"subdomain_ttl_days"`
	Status           int       `gorm:"not null;default:1" json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Subdomain struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	DomainID   uint       `gorm:"index;not null" json:"domain_id"`
	Name       string     `gorm:"size:128;not null" json:"name"`
	FullDomain string     `gorm:"size:255;uniqueIndex;not null" json:"full_domain"`
	Status     int        `gorm:"not null;default:1" json:"status"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (s *Subdomain) IsExpired() bool {
	return s.ExpiresAt != nil && s.ExpiresAt.Before(time.Now()) && s.Status == SubdomainStatusActive
}

type Record struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	DomainID    uint      `gorm:"index;not null" json:"domain_id"`
	SubdomainID uint      `gorm:"index;not null" json:"subdomain_id"`
	RemoteID    string    `gorm:"size:128" json:"remote_id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Type        string    `gorm:"size:16;not null" json:"type"`
	Value       string    `gorm:"size:1024;not null" json:"value"`
	TTL         int       `gorm:"not null;default:600" json:"ttl"`
	Line        string    `gorm:"size:64" json:"line"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PointsLedger struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Delta     int64     `gorm:"not null" json:"delta"`
	Balance   int64     `gorm:"not null" json:"balance"`
	Type      string    `gorm:"size:32;not null" json:"type"`
	BizNo     string    `gorm:"size:64;uniqueIndex" json:"biz_no"`
	Remark    string    `gorm:"size:512" json:"remark"`
	Operator  uint      `gorm:"default:0" json:"operator"`
	CreatedAt time.Time `json:"created_at"`
}

type RedeemCode struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Code      string     `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Points    int64      `gorm:"not null" json:"points"`
	MaxUses   int        `gorm:"not null;default:1" json:"max_uses"`
	UsedCount int        `gorm:"not null;default:0" json:"used_count"`
	ExpiresAt *time.Time `json:"expires_at"`
	Enabled   bool       `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type RedeemUse struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CodeID    uint      `gorm:"index;not null" json:"code_id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type PaymentOrder struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	OutTradeNo string     `gorm:"size:64;uniqueIndex;not null" json:"out_trade_no"`
	TradeNo    string     `gorm:"size:64;index" json:"trade_no"`
	AmountCent int64      `gorm:"not null" json:"amount_cent"`
	Points     int64      `gorm:"not null" json:"points"`
	Status     string     `gorm:"size:16;not null;default:pending" json:"status"`
	PayURL     string     `gorm:"size:1024" json:"pay_url"`
	NotifyRaw  string     `gorm:"type:text" json:"-"`
	PaidAt     *time.Time `json:"paid_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type Setting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"size:64;uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	Read      bool      `gorm:"not null;default:false" json:"read"`
	Link      string    `gorm:"size:512" json:"link"`
	CreatedAt time.Time `json:"created_at"`
}

type Webhook struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	URL       string    `gorm:"size:1024;not null" json:"url"`
	Events    string    `gorm:"size:512;not null" json:"events"`
	Secret    string    `gorm:"size:255" json:"-"`
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ApiKey struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	Name        string     `gorm:"size:128;not null" json:"name"`
	KeyPrefix   string     `gorm:"size:16;not null" json:"key_prefix"`
	KeyHash     string     `gorm:"size:255;not null" json:"-"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	LastIP      string     `gorm:"size:64" json:"last_ip"`
	Enabled     bool       `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type OperationLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index" json:"user_id"`
	AdminID    uint      `gorm:"index" json:"admin_id"`
	Action     string    `gorm:"size:64;not null" json:"action"`
	TargetType string    `gorm:"size:32" json:"target_type"`
	TargetID   string    `gorm:"size:64" json:"target_id"`
	IP         string    `gorm:"size:64" json:"ip"`
	Message    string    `gorm:"size:512" json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&Domain{},
		&Subdomain{},
		&Record{},
		&PointsLedger{},
		&RedeemCode{},
		&RedeemUse{},
		&PaymentOrder{},
		&Setting{},
		&OperationLog{},
		&Notification{},
		&Webhook{},
		&ApiKey{},
	}
}
