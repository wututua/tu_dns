package providers

import (
	"bytes"
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
	dns.Register("cloudflare", func() dns.Provider { return &Cloudflare{} })
}

type Cloudflare struct {
	apiToken string
	client   *http.Client
}

func (p *Cloudflare) Key() string   { return "cloudflare" }
func (p *Cloudflare) Label() string { return "Cloudflare" }

func (p *Cloudflare) ConfigFields() []dns.ConfigField {
	return []dns.ConfigField{
		{Name: "api_token", Label: "API Token", Required: true, Secret: true, Description: "Zone DNS Edit 权限 Token"},
	}
}

func (p *Cloudflare) Configure(config map[string]string) error {
	p.apiToken = strings.TrimSpace(config["api_token"])
	if p.apiToken == "" {
		return fmt.Errorf("api_token required")
	}
	p.client = &http.Client{Timeout: 20 * time.Second}
	return nil
}

func (p *Cloudflare) Check(ctx context.Context) error {
	_, err := p.ListZones(ctx)
	return err
}

func (p *Cloudflare) ListZones(ctx context.Context) ([]dns.Zone, error) {
	var result struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Result []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}
	if err := p.do(ctx, http.MethodGet, "https://api.cloudflare.com/client/v4/zones?per_page=50", nil, &result); err != nil {
		return nil, err
	}
	if !result.Success {
		msg := "cloudflare error"
		if len(result.Errors) > 0 {
			msg = result.Errors[0].Message
		}
		return nil, fmt.Errorf("%s", msg)
	}
	zones := make([]dns.Zone, 0, len(result.Result))
	for _, z := range result.Result {
		zones = append(zones, dns.Zone{ID: z.ID, Domain: z.Name})
	}
	return zones, nil
}

func (p *Cloudflare) CreateRecord(ctx context.Context, zone dns.Zone, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 1 // auto
	}
	body := map[string]interface{}{
		"type":    strings.ToUpper(input.Type),
		"name":    input.Name,
		"content": input.Value,
		"ttl":     ttl,
	}
	var result struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Result struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Type    string `json:"type"`
			Content string `json:"content"`
			TTL     int    `json:"ttl"`
		} `json:"result"`
	}
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zone.ID)
	if err := p.do(ctx, http.MethodPost, url, body, &result); err != nil {
		return dns.Record{}, err
	}
	if !result.Success {
		msg := "create failed"
		if len(result.Errors) > 0 {
			msg = result.Errors[0].Message
		}
		return dns.Record{}, fmt.Errorf("%s", msg)
	}
	return dns.Record{
		RemoteID: result.Result.ID,
		Name:     result.Result.Name,
		Type:     result.Result.Type,
		Value:    result.Result.Content,
		TTL:      result.Result.TTL,
	}, nil
}

func (p *Cloudflare) UpdateRecord(ctx context.Context, zone dns.Zone, remoteID string, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 1
	}
	body := map[string]interface{}{
		"type":    strings.ToUpper(input.Type),
		"name":    input.Name,
		"content": input.Value,
		"ttl":     ttl,
	}
	var result struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Result struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Type    string `json:"type"`
			Content string `json:"content"`
			TTL     int    `json:"ttl"`
		} `json:"result"`
	}
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zone.ID, remoteID)
	if err := p.do(ctx, http.MethodPut, url, body, &result); err != nil {
		return dns.Record{}, err
	}
	if !result.Success {
		msg := "update failed"
		if len(result.Errors) > 0 {
			msg = result.Errors[0].Message
		}
		return dns.Record{}, fmt.Errorf("%s", msg)
	}
	return dns.Record{
		RemoteID: result.Result.ID,
		Name:     result.Result.Name,
		Type:     result.Result.Type,
		Value:    result.Result.Content,
		TTL:      result.Result.TTL,
	}, nil
}

func (p *Cloudflare) DeleteRecord(ctx context.Context, zone dns.Zone, remoteID string) error {
	var result struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zone.ID, remoteID)
	if err := p.do(ctx, http.MethodDelete, url, nil, &result); err != nil {
		return err
	}
	if !result.Success {
		msg := "delete failed"
		if len(result.Errors) > 0 {
			msg = result.Errors[0].Message
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func (p *Cloudflare) do(ctx context.Context, method, url string, body interface{}, out interface{}) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiToken)
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
		return fmt.Errorf("cloudflare http %d: %s", resp.StatusCode, string(data))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}
