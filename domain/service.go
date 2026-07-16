package domain

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"tudns/config"
	"tudns/dns"
	_ "tudns/dns/providers"
	"tudns/models"
	"tudns/crypto"

	"gorm.io/gorm"
)

type Service struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

type SaveInput struct {
	Name         string            `json:"name"`
	ProviderKey  string            `json:"provider_key"`
	RemoteZoneID string            `json:"remote_zone_id"`
	Config       map[string]string `json:"config"`
	RecordTypes  string            `json:"record_types"`
	PointsCost   int64             `json:"points_cost"`
	Description  string            `json:"description"`
	Status       int               `json:"status"`
}

func (s *Service) ListPublic() ([]models.Domain, error) {
	var items []models.Domain
	err := s.db.Where("status = ?", models.DomainStatusActive).Order("id desc").Find(&items).Error
	return items, err
}

func (s *Service) ListAll() ([]models.Domain, error) {
	var items []models.Domain
	err := s.db.Order("id desc").Find(&items).Error
	return items, err
}

func (s *Service) Get(id uint) (*models.Domain, error) {
	var d models.Domain
	if err := s.db.First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Service) Save(id uint, in SaveInput) (*models.Domain, error) {
	name := strings.ToLower(strings.TrimSpace(in.Name))
	if name == "" || in.ProviderKey == "" {
		return nil, errors.New("域名和DNS平台必填")
	}
	if in.RecordTypes == "" {
		in.RecordTypes = "A,AAAA,CNAME,TXT"
	}
	if in.Status == 0 {
		in.Status = models.DomainStatusActive
	}

	var d models.Domain
	if id > 0 {
		if err := s.db.First(&d, id).Error; err != nil {
			return nil, err
		}
	}

	if len(in.Config) > 0 {
		raw, err := json.Marshal(in.Config)
		if err != nil {
			return nil, err
		}
		cipher, err := crypto.Encrypt(s.cfg.Security.SecretKey, string(raw))
		if err != nil {
			return nil, err
		}
		d.ConfigCiphertext = cipher
	} else if id == 0 {
		return nil, errors.New("DNS 平台配置必填")
	}

	d.Name = name
	d.ProviderKey = in.ProviderKey
	d.RemoteZoneID = in.RemoteZoneID
	d.RecordTypes = in.RecordTypes
	d.PointsCost = in.PointsCost
	d.Description = in.Description
	d.Status = in.Status

	if id == 0 {
		if err := s.db.Create(&d).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.Save(&d).Error; err != nil {
			return nil, err
		}
	}
	return &d, nil
}

func (s *Service) Delete(id uint) error {
	return s.db.Delete(&models.Domain{}, id).Error
}

func (s *Service) DecryptConfig(d *models.Domain) (map[string]string, error) {
	plain, err := crypto.Decrypt(s.cfg.Security.SecretKey, d.ConfigCiphertext)
	if err != nil {
		return nil, err
	}
	if plain == "" {
		return map[string]string{}, nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(plain), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Service) BuildProvider(d *models.Domain) (dns.Provider, dns.Zone, error) {
	p, ok := dns.New(d.ProviderKey)
	if !ok {
		return nil, dns.Zone{}, errors.New("未知 DNS 平台")
	}
	cfgMap, err := s.DecryptConfig(d)
	if err != nil {
		return nil, dns.Zone{}, err
	}
	if err := p.Configure(cfgMap); err != nil {
		return nil, dns.Zone{}, err
	}
	zone := dns.Zone{ID: d.RemoteZoneID, Domain: d.Name}
	return p, zone, nil
}

func (s *Service) CheckProvider(ctx context.Context, providerKey string, cfgMap map[string]string) error {
	p, ok := dns.New(providerKey)
	if !ok {
		return errors.New("未知 DNS 平台")
	}
	if err := p.Configure(cfgMap); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	return p.Check(ctx)
}

func (s *Service) ListZones(ctx context.Context, providerKey string, cfgMap map[string]string) ([]dns.Zone, error) {
	p, ok := dns.New(providerKey)
	if !ok {
		return nil, errors.New("未知 DNS 平台")
	}
	if err := p.Configure(cfgMap); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	return p.ListZones(ctx)
}

func (s *Service) AllowedTypes(d *models.Domain) map[string]bool {
	m := map[string]bool{}
	for _, t := range strings.Split(d.RecordTypes, ",") {
		t = strings.ToUpper(strings.TrimSpace(t))
		if t != "" {
			m[t] = true
		}
	}
	return m
}
