package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tudns/dns"
)

func init() {
	dns.Register("dnsla", func() dns.Provider { return &DNSLA{} })
}

type DNSLA struct {
	apiID  string
	apiKey string
	client *http.Client
}

func (p *DNSLA) Key() string   { return "dnsla" }
func (p *DNSLA) Label() string { return "DNS.LA" }

func (p *DNSLA) ConfigFields() []dns.ConfigField {
	return []dns.ConfigField{
		{Name: "api_id", Label: "API ID", Required: true, Secret: false},
		{Name: "api_key", Label: "API Key", Required: true, Secret: true},
	}
}

func (p *DNSLA) Configure(config map[string]string) error {
	p.apiID = strings.TrimSpace(config["api_id"])
	p.apiKey = strings.TrimSpace(config["api_key"])
	if p.apiID == "" || p.apiKey == "" {
		return fmt.Errorf("api_id and api_key required")
	}
	p.client = &http.Client{Timeout: 20 * time.Second}
	return nil
}

func (p *DNSLA) Check(ctx context.Context) error {
	_, err := p.ListZones(ctx)
	return err
}

func (p *DNSLA) post(ctx context.Context, path string, body map[string]interface{}, out interface{}) error {
	body["api_id"] = p.apiID
	body["api_key"] = p.apiKey

	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.dns.la/"+path, strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("dnsla http %d: %s", resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}

func (p *DNSLA) ListZones(ctx context.Context) ([]dns.Zone, error) {
	var result struct {
		Code int `json:"code"`
		Data []struct {
			ID     string `json:"id"`
			Domain string `json:"domain"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := p.post(ctx, "domain/list", map[string]interface{}{}, &result); err != nil {
		return nil, err
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("dnsla: %s", result.Msg)
	}
	zones := make([]dns.Zone, 0, len(result.Data))
	for _, z := range result.Data {
		zones = append(zones, dns.Zone{ID: z.ID, Domain: z.Domain})
	}
	return zones, nil
}

func (p *DNSLA) CreateRecord(ctx context.Context, zone dns.Zone, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	body := map[string]interface{}{
		"domain":    zone.Domain,
		"host":      hostToSub(input.Name, zone.Domain),
		"type":      strings.ToUpper(input.Type),
		"recordval": input.Value,
		"ttl":       ttl,
	}
	var result struct {
		Code int    `json:"code"`
		ID   string `json:"id"`
		Msg  string `json:"msg"`
	}
	if err := p.post(ctx, "record/add", body, &result); err != nil {
		return dns.Record{}, err
	}
	if result.Code != 200 {
		return dns.Record{}, fmt.Errorf("dnsla: %s", result.Msg)
	}
	return dns.Record{RemoteID: result.ID, Name: input.Name, Type: strings.ToUpper(input.Type), Value: input.Value, TTL: ttl}, nil
}

func (p *DNSLA) UpdateRecord(ctx context.Context, zone dns.Zone, remoteID string, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	body := map[string]interface{}{
		"domain":    zone.Domain,
		"id":        remoteID,
		"host":      hostToSub(input.Name, zone.Domain),
		"type":      strings.ToUpper(input.Type),
		"recordval": input.Value,
		"ttl":       ttl,
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := p.post(ctx, "record/edit", body, &result); err != nil {
		return dns.Record{}, err
	}
	if result.Code != 200 {
		return dns.Record{}, fmt.Errorf("dnsla: %s", result.Msg)
	}
	return dns.Record{RemoteID: remoteID, Name: input.Name, Type: strings.ToUpper(input.Type), Value: input.Value, TTL: ttl}, nil
}

func (p *DNSLA) DeleteRecord(ctx context.Context, zone dns.Zone, remoteID string) error {
	body := map[string]interface{}{
		"domain": zone.Domain,
		"id":     remoteID,
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := p.post(ctx, "record/del", body, &result); err != nil {
		return err
	}
	if result.Code != 200 {
		return fmt.Errorf("dnsla: %s", result.Msg)
	}
	return nil
}
