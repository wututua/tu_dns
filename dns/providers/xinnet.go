package providers

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tudns/dns"
)

func init() {
	dns.Register("xinnet", func() dns.Provider { return &Xinnet{} })
}

type Xinnet struct {
	apiKey    string
	secretKey string
	client    *http.Client
}

func (p *Xinnet) Key() string   { return "xinnet" }
func (p *Xinnet) Label() string { return "新网" }

func (p *Xinnet) ConfigFields() []dns.ConfigField {
	return []dns.ConfigField{
		{Name: "api_key", Label: "API Key", Required: true, Secret: false},
		{Name: "secret_key", Label: "Secret Key", Required: true, Secret: true},
	}
}

func (p *Xinnet) Configure(config map[string]string) error {
	p.apiKey = strings.TrimSpace(config["api_key"])
	p.secretKey = strings.TrimSpace(config["secret_key"])
	if p.apiKey == "" || p.secretKey == "" {
		return fmt.Errorf("api_key and secret_key required")
	}
	p.client = &http.Client{Timeout: 20 * time.Second}
	return nil
}

func (p *Xinnet) Check(ctx context.Context) error {
	_, err := p.ListZones(ctx)
	return err
}

func (p *Xinnet) sign() (timestamp, sign string) {
	timestamp = fmt.Sprintf("%d", time.Now().Unix())
	raw := p.apiKey + p.secretKey + timestamp
	sign = fmt.Sprintf("%x", md5.Sum([]byte(raw)))
	return
}

func (p *Xinnet) post(ctx context.Context, path string, body map[string]interface{}, out interface{}) error {
	ts, sig := p.sign()
	body["api_key"] = p.apiKey
	body["timestamp"] = ts
	body["sign"] = sig

	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.xinnet.com/dns/v2"+path, strings.NewReader(string(b)))
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
		return fmt.Errorf("xinnet http %d: %s", resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}

func (p *Xinnet) ListZones(ctx context.Context) ([]dns.Zone, error) {
	var result struct {
		Code int `json:"code"`
		Data []struct {
			Domain string `json:"domain"`
			ID     string `json:"id"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := p.post(ctx, "/domain/list", map[string]interface{}{}, &result); err != nil {
		return nil, err
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("xinnet: %s", result.Msg)
	}
	zones := make([]dns.Zone, 0, len(result.Data))
	for _, z := range result.Data {
		zones = append(zones, dns.Zone{ID: z.ID, Domain: z.Domain})
	}
	return zones, nil
}

func (p *Xinnet) CreateRecord(ctx context.Context, zone dns.Zone, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	body := map[string]interface{}{
		"domain_id": zone.ID,
		"host":      hostToSub(input.Name, zone.Domain),
		"type":      strings.ToUpper(input.Type),
		"value":     input.Value,
		"ttl":       ttl,
	}
	var result struct {
		Code int    `json:"code"`
		ID   string `json:"id"`
		Msg  string `json:"msg"`
	}
	if err := p.post(ctx, "/record/create", body, &result); err != nil {
		return dns.Record{}, err
	}
	if result.Code != 200 {
		return dns.Record{}, fmt.Errorf("xinnet: %s", result.Msg)
	}
	return dns.Record{RemoteID: result.ID, Name: input.Name, Type: strings.ToUpper(input.Type), Value: input.Value, TTL: ttl}, nil
}

func (p *Xinnet) UpdateRecord(ctx context.Context, zone dns.Zone, remoteID string, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	body := map[string]interface{}{
		"record_id": remoteID,
		"domain_id": zone.ID,
		"host":      hostToSub(input.Name, zone.Domain),
		"type":      strings.ToUpper(input.Type),
		"value":     input.Value,
		"ttl":       ttl,
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := p.post(ctx, "/record/update", body, &result); err != nil {
		return dns.Record{}, err
	}
	if result.Code != 200 {
		return dns.Record{}, fmt.Errorf("xinnet: %s", result.Msg)
	}
	return dns.Record{RemoteID: remoteID, Name: input.Name, Type: strings.ToUpper(input.Type), Value: input.Value, TTL: ttl}, nil
}

func (p *Xinnet) DeleteRecord(ctx context.Context, zone dns.Zone, remoteID string) error {
	body := map[string]interface{}{
		"record_id": remoteID,
		"domain_id": zone.ID,
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := p.post(ctx, "/record/delete", body, &result); err != nil {
		return err
	}
	if result.Code != 200 {
		return fmt.Errorf("xinnet: %s", result.Msg)
	}
	return nil
}
