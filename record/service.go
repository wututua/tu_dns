package record

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"tudns/dns"
	"tudns/domain"
	"tudns/models"
	"tudns/points"

	"gorm.io/gorm"
)

var (
	subNameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	reserved  = map[string]bool{
		"www": true, "mail": true, "ftp": true, "ns": true, "mx": true,
		"admin": true, "api": true, "cdn": true, "test": true,
	}
)

type Service struct {
	db     *gorm.DB
	domain *domain.Service
	points *points.Service
}

func NewService(db *gorm.DB, domainSvc *domain.Service, pointsSvc *points.Service) *Service {
	return &Service{db: db, domain: domainSvc, points: pointsSvc}
}

// CreateBundle creates subdomain + first record in one step, charging once.
type BundleInput struct {
	DomainID      uint   `json:"domain_id"`
	SubdomainName string `json:"subdomain_name"`
	Type          string `json:"type"`
	Value         string `json:"value"`
	TTL           int    `json:"ttl"`
	Line          string `json:"line"`
}

type BundleResult struct {
	Subdomain models.Subdomain `json:"subdomain"`
	Record    models.Record    `json:"record"`
	Charged   int64            `json:"charged"`
}

func (s *Service) CreateBundle(userID uint, in BundleInput) (*BundleResult, error) {
	name := strings.ToLower(strings.TrimSpace(in.SubdomainName))
	if !subNameRe.MatchString(name) {
		return nil, errors.New("子域名前缀格式无效")
	}
	if reserved[name] {
		return nil, errors.New("该子域名为保留字")
	}
	recType := strings.ToUpper(strings.TrimSpace(in.Type))
	value := strings.TrimSpace(in.Value)
	if err := validateRecord(recType, value); err != nil {
		return nil, err
	}

	d, err := s.domain.Get(in.DomainID)
	if err != nil {
		return nil, errors.New("域名不存圀")
	}
	if d.Status != models.DomainStatusActive {
		return nil, errors.New("域名未上枀")
	}
	if !s.domain.AllowedTypes(d)[recType] {
		return nil, fmt.Errorf("该域名不允许 %s 记录", recType)
	}

	full := name + "." + d.Name
	var exist int64
	if err := s.db.Model(&models.Subdomain{}).Where("full_domain = ?", full).Count(&exist).Error; err != nil {
		return nil, err
	}
	if exist > 0 {
		return nil, errors.New("子域名已被占甀")
	}

	provider, zone, err := s.domain.BuildProvider(d)
	if err != nil {
		return nil, err
	}

	ttl := in.TTL
	if ttl <= 0 {
		ttl = 600
	}

	var expiresAt *time.Time
	if d.SubdomainTTLDays > 0 {
		t := time.Now().Add(time.Duration(d.SubdomainTTLDays) * 24 * time.Hour)
		expiresAt = &t
	}

	var result BundleResult
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if d.PointsCost > 0 {
			if _, err := s.points.Charge(tx, userID, d.PointsCost, fmt.Sprintf("申请 %s 并添劀%s 记录", full, recType)); err != nil {
				return err
			}
		}
		result.Charged = d.PointsCost

		sub := models.Subdomain{
			UserID:     userID,
			DomainID:   d.ID,
			Name:       name,
			FullDomain: full,
			Status:     models.SubdomainStatusActive,
			ExpiresAt:  expiresAt,
		}
		if err := tx.Create(&sub).Error; err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		remote, err := provider.CreateRecord(ctx, zone, dns.RecordInput{
			Name:  full,
			Type:  recType,
			Value: value,
			TTL:   ttl,
			Line:  in.Line,
		})
		if err != nil {
			return fmt.Errorf("上游 DNS 创建失败: %w", err)
		}

		rec := models.Record{
			UserID:      userID,
			DomainID:    d.ID,
			SubdomainID: sub.ID,
			RemoteID:    remote.RemoteID,
			Name:        full,
			Type:        recType,
			Value:       value,
			TTL:         ttl,
			Line:        in.Line,
		}
		if err := tx.Create(&rec).Error; err != nil {
			// best-effort cleanup remote
			_ = provider.DeleteRecord(ctx, zone, remote.RemoteID)
			return err
		}
		result.Subdomain = sub
		result.Record = rec
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

type AddRecordInput struct {
	SubdomainID uint   `json:"subdomain_id"`
	Type        string `json:"type"`
	Value       string `json:"value"`
	TTL         int    `json:"ttl"`
	Line        string `json:"line"`
}

func (s *Service) checkExpired(sub *models.Subdomain) error {
	if sub.IsExpired() {
		s.db.Model(sub).Update("status", models.SubdomainStatusDisabled)
		return errors.New("子域名已过期")
	}
	return nil
}

func (s *Service) AddRecord(userID uint, in AddRecordInput, isAdmin bool) (*models.Record, int64, error) {
	var sub models.Subdomain
	if err := s.db.First(&sub, in.SubdomainID).Error; err != nil {
		return nil, 0, errors.New("子域名不存在")
	}
	if !isAdmin && sub.UserID != userID {
		return nil, 0, errors.New("无权操作")
	}
	if sub.Status != models.SubdomainStatusActive {
		return nil, 0, errors.New("子域名已禁用")
	}
	if err := s.checkExpired(&sub); err != nil {
		return nil, 0, err
	}
	d, err := s.domain.Get(sub.DomainID)
	if err != nil {
		return nil, 0, err
	}
	recType := strings.ToUpper(strings.TrimSpace(in.Type))
	value := strings.TrimSpace(in.Value)
	if err := validateRecord(recType, value); err != nil {
		return nil, 0, err
	}
	if !s.domain.AllowedTypes(d)[recType] {
		return nil, 0, fmt.Errorf("该域名不允许 %s 记录", recType)
	}
	provider, zone, err := s.domain.BuildProvider(d)
	if err != nil {
		return nil, 0, err
	}
	ttl := in.TTL
	if ttl <= 0 {
		ttl = 600
	}

	var rec models.Record
	var charged int64
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if d.PointsCost > 0 {
			if _, err := s.points.Charge(tx, userID, d.PointsCost, fmt.Sprintf("新增 %s %s 记录", sub.FullDomain, recType)); err != nil {
				return err
			}
			charged = d.PointsCost
		}
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		remote, err := provider.CreateRecord(ctx, zone, dns.RecordInput{
			Name:  sub.FullDomain,
			Type:  recType,
			Value: value,
			TTL:   ttl,
			Line:  in.Line,
		})
		if err != nil {
			return fmt.Errorf("上游 DNS 创建失败: %w", err)
		}
		rec = models.Record{
			UserID:      sub.UserID,
			DomainID:    d.ID,
			SubdomainID: sub.ID,
			RemoteID:    remote.RemoteID,
			Name:        sub.FullDomain,
			Type:        recType,
			Value:       value,
			TTL:         ttl,
			Line:        in.Line,
		}
		return tx.Create(&rec).Error
	})
	if err != nil {
		return nil, 0, err
	}
	return &rec, charged, nil
}

type UpdateRecordInput struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
	Line  string `json:"line"`
}

func (s *Service) UpdateRecord(userID, recordID uint, in UpdateRecordInput, isAdmin bool) (*models.Record, error) {
	var rec models.Record
	if err := s.db.First(&rec, recordID).Error; err != nil {
		return nil, errors.New("记录不存圀")
	}
	if !isAdmin && rec.UserID != userID {
		return nil, errors.New("无权操作")
	}
	var sub models.Subdomain
	if err := s.db.First(&sub, rec.SubdomainID).Error; err == nil {
		if err := s.checkExpired(&sub); err != nil {
			return nil, err
		}
	}
	d, err := s.domain.Get(rec.DomainID)
	if err != nil {
		return nil, err
	}
	recType := strings.ToUpper(strings.TrimSpace(in.Type))
	if recType == "" {
		recType = rec.Type
	}
	value := strings.TrimSpace(in.Value)
	if value == "" {
		value = rec.Value
	}
	if err := validateRecord(recType, value); err != nil {
		return nil, err
	}
	if !s.domain.AllowedTypes(d)[recType] {
		return nil, fmt.Errorf("该域名不允许 %s 记录", recType)
	}
	ttl := in.TTL
	if ttl <= 0 {
		ttl = rec.TTL
	}
	if ttl <= 0 {
		ttl = 600
	}
	line := in.Line
	if line == "" {
		line = rec.Line
	}

	provider, zone, err := s.domain.BuildProvider(d)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	remote, err := provider.UpdateRecord(ctx, zone, rec.RemoteID, dns.RecordInput{
		Name:  rec.Name,
		Type:  recType,
		Value: value,
		TTL:   ttl,
		Line:  line,
	})
	if err != nil {
		return nil, fmt.Errorf("上游 DNS 更新失败: %w", err)
	}
	rec.Type = recType
	rec.Value = value
	rec.TTL = ttl
	rec.Line = line
	if remote.RemoteID != "" {
		rec.RemoteID = remote.RemoteID
	}
	if err := s.db.Save(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *Service) DeleteRecord(userID, recordID uint, isAdmin bool) error {
	var rec models.Record
	if err := s.db.First(&rec, recordID).Error; err != nil {
		return errors.New("记录不存圀")
	}
	if !isAdmin && rec.UserID != userID {
		return errors.New("无权操作")
	}
	var sub models.Subdomain
	if err := s.db.First(&sub, rec.SubdomainID).Error; err == nil {
		if err := s.checkExpired(&sub); err != nil {
			return err
		}
	}
	d, err := s.domain.Get(rec.DomainID)
	if err != nil {
		return err
	}
	provider, zone, err := s.domain.BuildProvider(d)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if rec.RemoteID != "" {
		if err := provider.DeleteRecord(ctx, zone, rec.RemoteID); err != nil {
			return fmt.Errorf("上游 DNS 删除失败: %w", err)
		}
	}
	return s.db.Delete(&rec).Error
}

func (s *Service) ListByUser(userID uint) ([]models.Record, error) {
	var items []models.Record
	err := s.db.Where("user_id = ?", userID).Order("id desc").Find(&items).Error
	return items, err
}

func (s *Service) ListAll() ([]models.Record, error) {
	var items []models.Record
	err := s.db.Order("id desc").Find(&items).Error
	return items, err
}

func (s *Service) ListSubdomains(userID uint) ([]models.Subdomain, error) {
	var items []models.Subdomain
	q := s.db.Order("id desc")
	if userID > 0 {
		q = q.Where("user_id = ?", userID)
	}
	err := q.Find(&items).Error
	return items, err
}

func (s *Service) DeleteSubdomain(userID, subID uint, isAdmin bool) error {
	var sub models.Subdomain
	if err := s.db.First(&sub, subID).Error; err != nil {
		return errors.New("子域名不存在")
	}
	if !isAdmin && sub.UserID != userID {
		return errors.New("无权操作")
	}
	if err := s.checkExpired(&sub); err != nil && !isAdmin {
		return err
	}
	var records []models.Record
	if err := s.db.Where("subdomain_id = ?", sub.ID).Find(&records).Error; err != nil {
		return err
	}
	d, err := s.domain.Get(sub.DomainID)
	if err != nil {
		return err
	}
	provider, zone, err := s.domain.BuildProvider(d)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, rec := range records {
			if rec.RemoteID != "" {
				_ = provider.DeleteRecord(ctx, zone, rec.RemoteID)
			}
			if err := tx.Delete(&rec).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&sub).Error
	})
}

func validateRecord(recType, value string) error {
	if recType == "" || value == "" {
		return errors.New("记录类型和值必塀")
	}
	switch recType {
	case "A":
		ip := net.ParseIP(value)
		if ip == nil || ip.To4() == nil {
			return errors.New("A 记录值必须是 IPv4")
		}
	case "AAAA":
		ip := net.ParseIP(value)
		if ip == nil || ip.To4() != nil {
			return errors.New("AAAA 记录值必须是 IPv6")
		}
	case "CNAME", "TXT", "MX", "NS":
		if len(value) > 1024 {
			return errors.New("记录值过镀")
		}
	default:
		return fmt.Errorf("不支持的记录类型: %s", recType)
	}
	return nil
}
