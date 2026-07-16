package providers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"tudns/dns"

	volcdns "github.com/volcengine/volc-sdk-golang/service/dns"
)

func init() {
	dns.Register("volcengine", func() dns.Provider { return &Volcengine{} })
}

type volcDNSClient interface {
	ListZones(context.Context, *volcdns.ListZonesRequest) (*volcdns.ListZonesResponse, error)
	CreateRecord(context.Context, *volcdns.CreateRecordRequest) (*volcdns.CreateRecordResponse, error)
	UpdateRecord(context.Context, *volcdns.UpdateRecordRequest) (*volcdns.UpdateRecordResponse, error)
	DeleteRecord(context.Context, *volcdns.DeleteRecordRequest) error
	QueryRecord(context.Context, *volcdns.QueryRecordRequest) (*volcdns.QueryRecordResponse, error)
}

type Volcengine struct {
	client volcDNSClient
}

func (p *Volcengine) Key() string   { return "volcengine" }
func (p *Volcengine) Label() string { return "火山引擎 DNS" }

func (p *Volcengine) ConfigFields() []dns.ConfigField {
	return []dns.ConfigField{
		{Name: "access_key", Label: "Access Key", Required: true},
		{Name: "secret_key", Label: "Secret Key", Required: true, Secret: true},
	}
}

func (p *Volcengine) Configure(config map[string]string) error {
	ak := strings.TrimSpace(config["access_key"])
	sk := strings.TrimSpace(config["secret_key"])
	if ak == "" || sk == "" {
		return errors.New("access_key and secret_key required")
	}
	caller := volcdns.NewVolcCaller()
	caller.Volc.SetAccessKey(ak)
	caller.Volc.SetSecretKey(sk)
	caller.Volc.SetScheme("https")
	p.client = volcdns.NewClient(caller)
	return nil
}

func (p *Volcengine) Check(ctx context.Context) error {
	_, err := p.ListZones(ctx)
	return err
}

func (p *Volcengine) ListZones(ctx context.Context) ([]dns.Zone, error) {
	if err := p.ready(); err != nil {
		return nil, err
	}
	pageSize := "100"
	result, err := p.client.ListZones(ctx, &volcdns.ListZonesRequest{PageSize: &pageSize})
	if err != nil {
		return nil, fmt.Errorf("volcengine list zones: %w", err)
	}
	zones := make([]dns.Zone, 0, len(result.Zones))
	for _, zone := range result.Zones {
		if zone.ZID == nil || zone.ZoneName == nil {
			continue
		}
		zones = append(zones, dns.Zone{ID: strconv.FormatInt(*zone.ZID, 10), Domain: strings.TrimSuffix(*zone.ZoneName, ".")})
	}
	return zones, nil
}

func (p *Volcengine) CreateRecord(ctx context.Context, zone dns.Zone, input dns.RecordInput) (dns.Record, error) {
	if err := p.ready(); err != nil {
		return dns.Record{}, err
	}
	zid, err := strconv.ParseInt(zone.ID, 10, 64)
	if err != nil {
		return dns.Record{}, fmt.Errorf("volcengine invalid zone id: %w", err)
	}
	host := hostToSub(input.Name, zone.Domain)
	recordType := strings.ToUpper(input.Type)
	ttl := int64(normalizedTTL(input.TTL))
	line := input.Line
	request := &volcdns.CreateRecordRequest{ZID: &zid, Host: &host, Type: &recordType, Value: &input.Value, TTL: &ttl}
	if line != "" {
		request.Line = &line
	}
	result, err := p.client.CreateRecord(ctx, request)
	if err != nil {
		return dns.Record{}, fmt.Errorf("volcengine create record: %w", err)
	}
	if result.RecordID == nil || *result.RecordID == "" {
		return dns.Record{}, errors.New("volcengine create record returned no id")
	}
	return dns.Record{RemoteID: *result.RecordID, Name: input.Name, Type: recordType, Value: input.Value, TTL: int(ttl), Line: line}, nil
}

func (p *Volcengine) UpdateRecord(ctx context.Context, zone dns.Zone, remoteID string, input dns.RecordInput) (dns.Record, error) {
	if err := p.ready(); err != nil {
		return dns.Record{}, err
	}
	host := hostToSub(input.Name, zone.Domain)
	recordType := strings.ToUpper(input.Type)
	ttl := int64(normalizedTTL(input.TTL))
	line := input.Line
	if line == "" {
		result, err := p.client.QueryRecord(ctx, &volcdns.QueryRecordRequest{RecordID: &remoteID})
		if err != nil {
			return dns.Record{}, fmt.Errorf("volcengine query record line: %w", err)
		}
		if result.Line == nil || *result.Line == "" {
			return dns.Record{}, errors.New("volcengine query record returned no line")
		}
		line = *result.Line
	}
	_, err := p.client.UpdateRecord(ctx, &volcdns.UpdateRecordRequest{
		Host: host, Line: line, RecordID: remoteID, TTL: &ttl, Type: &recordType, Value: &input.Value,
	})
	if err != nil {
		return dns.Record{}, fmt.Errorf("volcengine update record: %w", err)
	}
	return dns.Record{RemoteID: remoteID, Name: input.Name, Type: recordType, Value: input.Value, TTL: int(ttl), Line: line}, nil
}

func (p *Volcengine) DeleteRecord(ctx context.Context, zone dns.Zone, remoteID string) error {
	if err := p.ready(); err != nil {
		return err
	}
	if err := p.client.DeleteRecord(ctx, &volcdns.DeleteRecordRequest{RecordID: &remoteID}); err != nil {
		return fmt.Errorf("volcengine delete record: %w", err)
	}
	return nil
}

func (p *Volcengine) ready() error {
	if p.client == nil {
		return errors.New("volcengine dns is not configured")
	}
	return nil
}
