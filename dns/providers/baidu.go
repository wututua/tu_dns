package providers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tudns/dns"

	baidudns "github.com/baidubce/bce-sdk-go/services/dns"
	"github.com/google/uuid"
)

func init() {
	dns.Register("baidu", func() dns.Provider { return &BaiduCloud{} })
}

type baiduDNSClient interface {
	ListZone(*baidudns.ListZoneRequest) (*baidudns.ListZoneResponse, error)
	ListRecord(string, *baidudns.ListRecordRequest) (*baidudns.ListRecordResponse, error)
	CreateRecord(string, *baidudns.CreateRecordRequest, string) error
	UpdateRecord(string, string, *baidudns.UpdateRecordRequest, string) error
	DeleteRecord(string, string, string) error
}

type BaiduCloud struct {
	client baiduDNSClient
}

func (p *BaiduCloud) Key() string   { return "baidu" }
func (p *BaiduCloud) Label() string { return "度度智能云DNS" }

func (p *BaiduCloud) ConfigFields() []dns.ConfigField {
	return []dns.ConfigField{
		{Name: "access_key", Label: "Access Key", Required: true},
		{Name: "secret_key", Label: "Secret Key", Required: true, Secret: true},
	}
}

func (p *BaiduCloud) Configure(config map[string]string) error {
	ak := strings.TrimSpace(config["access_key"])
	sk := strings.TrimSpace(config["secret_key"])
	if ak == "" || sk == "" {
		return errors.New("access_key and secret_key required")
	}
	client, err := baidudns.NewClient(ak, sk, "https://dns.baidubce.com")
	if err != nil {
		return fmt.Errorf("baidu dns client: %w", err)
	}
	p.client = client
	return nil
}

func (p *BaiduCloud) Check(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := p.ListZones(ctx)
	return err
}

func (p *BaiduCloud) ListZones(ctx context.Context) ([]dns.Zone, error) {
	if err := p.ready(ctx); err != nil {
		return nil, err
	}
	result, err := p.client.ListZone(&baidudns.ListZoneRequest{MaxKeys: 1000})
	if err != nil {
		return nil, fmt.Errorf("baidu list zones: %w", err)
	}
	zones := make([]dns.Zone, 0, len(result.Zones))
	for _, zone := range result.Zones {
		zones = append(zones, dns.Zone{ID: zone.Id, Domain: strings.TrimSuffix(zone.Name, ".")})
	}
	return zones, nil
}

func (p *BaiduCloud) CreateRecord(ctx context.Context, zone dns.Zone, input dns.RecordInput) (dns.Record, error) {
	if err := p.ready(ctx); err != nil {
		return dns.Record{}, err
	}
	ttl := normalizedTTL(input.TTL)
	ttl32 := int32(ttl)
	line := input.Line
	request := &baidudns.CreateRecordRequest{
		Rr: hostToSub(input.Name, zone.Domain), Type: strings.ToUpper(input.Type),
		Value: input.Value, Ttl: &ttl32,
	}
	if line != "" {
		request.Line = &line
	}
	before, err := p.matchingRecordIDs(zone, input)
	if err != nil {
		return dns.Record{}, err
	}
	if err := p.client.CreateRecord(zone.Domain, request, uuid.NewString()); err != nil {
		return dns.Record{}, fmt.Errorf("baidu create record: %w", err)
	}
	remoteID, err := p.findNewRecordID(ctx, zone, input, before)
	if err != nil {
		return dns.Record{}, err
	}
	return dns.Record{RemoteID: remoteID, Name: input.Name, Type: request.Type, Value: input.Value, TTL: ttl, Line: line}, nil
}

func (p *BaiduCloud) UpdateRecord(ctx context.Context, zone dns.Zone, remoteID string, input dns.RecordInput) (dns.Record, error) {
	if err := p.ready(ctx); err != nil {
		return dns.Record{}, err
	}
	ttl := normalizedTTL(input.TTL)
	ttl32 := int32(ttl)
	line := input.Line
	request := &baidudns.UpdateRecordRequest{
		Rr: hostToSub(input.Name, zone.Domain), Type: strings.ToUpper(input.Type),
		Value: input.Value, Ttl: &ttl32,
	}
	if err := p.client.UpdateRecord(zone.Domain, remoteID, request, uuid.NewString()); err != nil {
		return dns.Record{}, fmt.Errorf("baidu update record: %w", err)
	}
	return dns.Record{RemoteID: remoteID, Name: input.Name, Type: request.Type, Value: input.Value, TTL: ttl, Line: line}, nil
}

func (p *BaiduCloud) DeleteRecord(ctx context.Context, zone dns.Zone, remoteID string) error {
	if err := p.ready(ctx); err != nil {
		return err
	}
	if err := p.client.DeleteRecord(zone.Domain, remoteID, uuid.NewString()); err != nil {
		return fmt.Errorf("baidu delete record: %w", err)
	}
	return nil
}

func (p *BaiduCloud) matchingRecordIDs(zone dns.Zone, input dns.RecordInput) (map[string]struct{}, error) {
	rr := hostToSub(input.Name, zone.Domain)
	result, err := p.client.ListRecord(zone.Domain, &baidudns.ListRecordRequest{Rr: rr, MaxKeys: 1000})
	if err != nil {
		return nil, fmt.Errorf("baidu list matching records: %w", err)
	}
	ids := make(map[string]struct{})
	for _, record := range result.Records {
		if record.Rr == rr && strings.EqualFold(record.Type, input.Type) && record.Value == input.Value {
			ids[record.Id] = struct{}{}
		}
	}
	return ids, nil
}

func (p *BaiduCloud) findNewRecordID(ctx context.Context, zone dns.Zone, input dns.RecordInput, before map[string]struct{}) (string, error) {
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		}
		after, err := p.matchingRecordIDs(zone, input)
		if err != nil {
			return "", err
		}
		var remoteID string
		for id := range after {
			if _, existed := before[id]; existed {
				continue
			}
			if remoteID != "" {
				return "", errors.New("baidu created multiple matching records; record id is ambiguous")
			}
			remoteID = id
		}
		if remoteID != "" {
			return remoteID, nil
		}
	}
	return "", errors.New("baidu created record id not found")
}

func (p *BaiduCloud) ready(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.client == nil {
		return errors.New("baidu dns is not configured")
	}
	return nil
}

func normalizedTTL(ttl int) int {
	if ttl <= 0 {
		return 600
	}
	return ttl
}
