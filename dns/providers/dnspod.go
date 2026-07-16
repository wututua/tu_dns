package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tudns/dns"
)

func init() {
	dns.Register("dnspod", func() dns.Provider { return &DNSPod{} })
}

// DNSPod uses Tencent Cloud DNSPod API (API Token).
type DNSPod struct {
	secretID  string
	secretKey string
	client    *http.Client
}

func (p *DNSPod) Key() string   { return "dnspod" }
func (p *DNSPod) Label() string { return "DNSPod" }

func (p *DNSPod) ConfigFields() []dns.ConfigField {
	return []dns.ConfigField{
		{Name: "id", Label: "Token ID", Required: true, Secret: false},
		{Name: "token", Label: "Token", Required: true, Secret: true},
	}
}

func (p *DNSPod) Configure(config map[string]string) error {
	p.secretID = strings.TrimSpace(config["id"])
	p.secretKey = strings.TrimSpace(config["token"])
	if p.secretID == "" || p.secretKey == "" {
		return fmt.Errorf("id and token required")
	}
	p.client = &http.Client{Timeout: 20 * time.Second}
	return nil
}

func (p *DNSPod) loginToken() string {
	return p.secretID + "," + p.secretKey
}

func (p *DNSPod) Check(ctx context.Context) error {
	_, err := p.ListZones(ctx)
	return err
}

func (p *DNSPod) ListZones(ctx context.Context) ([]dns.Zone, error) {
	form := url.Values{}
	form.Set("login_token", p.loginToken())
	form.Set("format", "json")
	var result struct {
		Status struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"status"`
		Domains []struct {
			ID   json.Number `json:"id"`
			Name string      `json:"name"`
		} `json:"domains"`
	}
	if err := p.post(ctx, "https://dnsapi.cn/Domain.List", form, &result); err != nil {
		return nil, err
	}
	if result.Status.Code != "1" {
		return nil, fmt.Errorf("dnspod: %s", result.Status.Message)
	}
	zones := make([]dns.Zone, 0, len(result.Domains))
	for _, d := range result.Domains {
		zones = append(zones, dns.Zone{ID: d.ID.String(), Domain: d.Name})
	}
	return zones, nil
}

func (p *DNSPod) CreateRecord(ctx context.Context, zone dns.Zone, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	sub := hostToSub(input.Name, zone.Domain)
	form := url.Values{}
	form.Set("login_token", p.loginToken())
	form.Set("format", "json")
	form.Set("domain_id", zone.ID)
	form.Set("sub_domain", sub)
	form.Set("record_type", strings.ToUpper(input.Type))
	form.Set("record_line", "默认")
	if input.Line != "" {
		form.Set("record_line", input.Line)
	}
	form.Set("value", input.Value)
	form.Set("ttl", strconv.Itoa(ttl))
	var result struct {
		Status struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"status"`
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	if err := p.post(ctx, "https://dnsapi.cn/Record.Create", form, &result); err != nil {
		return dns.Record{}, err
	}
	if result.Status.Code != "1" {
		return dns.Record{}, fmt.Errorf("dnspod: %s", result.Status.Message)
	}
	return dns.Record{
		RemoteID: result.Record.ID,
		Name:     input.Name,
		Type:     strings.ToUpper(input.Type),
		Value:    input.Value,
		TTL:      ttl,
	}, nil
}

func (p *DNSPod) UpdateRecord(ctx context.Context, zone dns.Zone, remoteID string, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	sub := hostToSub(input.Name, zone.Domain)
	form := url.Values{}
	form.Set("login_token", p.loginToken())
	form.Set("format", "json")
	form.Set("domain_id", zone.ID)
	form.Set("record_id", remoteID)
	form.Set("sub_domain", sub)
	form.Set("record_type", strings.ToUpper(input.Type))
	form.Set("record_line", "默认")
	if input.Line != "" {
		form.Set("record_line", input.Line)
	}
	form.Set("value", input.Value)
	form.Set("ttl", strconv.Itoa(ttl))
	var result struct {
		Status struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"status"`
	}
	if err := p.post(ctx, "https://dnsapi.cn/Record.Modify", form, &result); err != nil {
		return dns.Record{}, err
	}
	if result.Status.Code != "1" {
		return dns.Record{}, fmt.Errorf("dnspod: %s", result.Status.Message)
	}
	return dns.Record{
		RemoteID: remoteID,
		Name:     input.Name,
		Type:     strings.ToUpper(input.Type),
		Value:    input.Value,
		TTL:      ttl,
	}, nil
}

func (p *DNSPod) DeleteRecord(ctx context.Context, zone dns.Zone, remoteID string) error {
	form := url.Values{}
	form.Set("login_token", p.loginToken())
	form.Set("format", "json")
	form.Set("domain_id", zone.ID)
	form.Set("record_id", remoteID)
	var result struct {
		Status struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"status"`
	}
	if err := p.post(ctx, "https://dnsapi.cn/Record.Remove", form, &result); err != nil {
		return err
	}
	if result.Status.Code != "1" {
		return fmt.Errorf("dnspod: %s", result.Status.Message)
	}
	return nil
}

func (p *DNSPod) post(ctx context.Context, endpoint string, form url.Values, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
		return fmt.Errorf("dnspod http %d: %s", resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}

func hostToSub(name, zone string) string {
	name = strings.TrimSuffix(strings.ToLower(name), ".")
	zone = strings.TrimSuffix(strings.ToLower(zone), ".")
	if name == zone {
		return "@"
	}
	suffix := "." + zone
	if strings.HasSuffix(name, suffix) {
		return strings.TrimSuffix(name, suffix)
	}
	return name
}
