package providers

import (
	"context"
	"crypto/md5"
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
	dns.Register("westcn", func() dns.Provider { return &WestCn{} })
}

type WestCn struct {
	username    string
	apiPassword string
	client      *http.Client
}

func (p *WestCn) Key() string   { return "westcn" }
func (p *WestCn) Label() string { return "西部数码" }

func (p *WestCn) ConfigFields() []dns.ConfigField {
	return []dns.ConfigField{
		{Name: "username", Label: "用户名", Required: true, Secret: false},
		{Name: "api_password", Label: "API 密码", Required: true, Secret: true},
	}
}

func (p *WestCn) Configure(config map[string]string) error {
	p.username = strings.TrimSpace(config["username"])
	p.apiPassword = strings.TrimSpace(config["api_password"])
	if p.username == "" || p.apiPassword == "" {
		return fmt.Errorf("username and api_password required")
	}
	p.client = &http.Client{Timeout: 20 * time.Second}
	return nil
}

func (p *WestCn) Check(ctx context.Context) error {
	_, err := p.ListZones(ctx)
	return err
}

func (p *WestCn) apiToken() string {
	raw := p.username + p.apiPassword + time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}

func (p *WestCn) post(ctx context.Context, action string, form url.Values, out interface{}) error {
	form.Set("username", p.username)
	form.Set("time", time.Now().UTC().Format("20060102150405"))
	form.Set("token", p.apiToken())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.west.cn/api/v2/dns/"+action, strings.NewReader(form.Encode()))
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
		return fmt.Errorf("westcn http %d: %s", resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}

func (p *WestCn) ListZones(ctx context.Context) ([]dns.Zone, error) {
	form := url.Values{}
	var result struct {
		Code int `json:"code"`
		Data []struct {
			Domain string `json:"domain"`
			ID     string `json:"id"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := p.post(ctx, "list", form, &result); err != nil {
		return nil, err
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("westcn: %s", result.Msg)
	}
	zones := make([]dns.Zone, 0, len(result.Data))
	for _, z := range result.Data {
		zones = append(zones, dns.Zone{ID: z.ID, Domain: z.Domain})
	}
	return zones, nil
}

func (p *WestCn) CreateRecord(ctx context.Context, zone dns.Zone, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	form := url.Values{}
	form.Set("domain", zone.Domain)
	form.Set("subdomain", hostToSub(input.Name, zone.Domain))
	form.Set("type", strings.ToUpper(input.Type))
	form.Set("value", input.Value)
	form.Set("ttl", strconv.Itoa(ttl))
	if input.Line != "" {
		form.Set("line", input.Line)
	}

	var result struct {
		Code int    `json:"code"`
		ID   string `json:"id"`
		Msg  string `json:"msg"`
	}
	if err := p.post(ctx, "addrecord", form, &result); err != nil {
		return dns.Record{}, err
	}
	if result.Code != 200 {
		return dns.Record{}, fmt.Errorf("westcn: %s", result.Msg)
	}
	return dns.Record{RemoteID: result.ID, Name: input.Name, Type: strings.ToUpper(input.Type), Value: input.Value, TTL: ttl}, nil
}

func (p *WestCn) UpdateRecord(ctx context.Context, zone dns.Zone, remoteID string, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	form := url.Values{}
	form.Set("id", remoteID)
	form.Set("domain", zone.Domain)
	form.Set("subdomain", hostToSub(input.Name, zone.Domain))
	form.Set("type", strings.ToUpper(input.Type))
	form.Set("value", input.Value)
	form.Set("ttl", strconv.Itoa(ttl))

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := p.post(ctx, "modrecord", form, &result); err != nil {
		return dns.Record{}, err
	}
	if result.Code != 200 {
		return dns.Record{}, fmt.Errorf("westcn: %s", result.Msg)
	}
	return dns.Record{RemoteID: remoteID, Name: input.Name, Type: strings.ToUpper(input.Type), Value: input.Value, TTL: ttl}, nil
}

func (p *WestCn) DeleteRecord(ctx context.Context, zone dns.Zone, remoteID string) error {
	form := url.Values{}
	form.Set("id", remoteID)
	form.Set("domain", zone.Domain)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := p.post(ctx, "delrecord", form, &result); err != nil {
		return err
	}
	if result.Code != 200 {
		return fmt.Errorf("westcn: %s", result.Msg)
	}
	return nil
}
