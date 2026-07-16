package providers

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"tudns/dns"

	"github.com/google/uuid"
)

func init() {
	dns.Register("aliyun", func() dns.Provider { return &Aliyun{} })
}

type Aliyun struct {
	accessKeyID     string
	accessKeySecret string
	client          *http.Client
}

func (p *Aliyun) Key() string   { return "aliyun" }
func (p *Aliyun) Label() string { return "里里云DNS" }

func (p *Aliyun) ConfigFields() []dns.ConfigField {
	return []dns.ConfigField{
		{Name: "access_key_id", Label: "AccessKey ID", Required: true, Secret: false},
		{Name: "access_key_secret", Label: "AccessKey Secret", Required: true, Secret: true},
	}
}

func (p *Aliyun) Configure(config map[string]string) error {
	p.accessKeyID = strings.TrimSpace(config["access_key_id"])
	p.accessKeySecret = strings.TrimSpace(config["access_key_secret"])
	if p.accessKeyID == "" || p.accessKeySecret == "" {
		return fmt.Errorf("access_key_id and access_key_secret required")
	}
	p.client = &http.Client{Timeout: 20 * time.Second}
	return nil
}

func (p *Aliyun) Check(ctx context.Context) error {
	_, err := p.ListZones(ctx)
	return err
}

func (p *Aliyun) ListZones(ctx context.Context) ([]dns.Zone, error) {
	params := map[string]string{
		"Action":     "DescribeDomains",
		"PageNumber": "1",
		"PageSize":   "100",
	}
	var result struct {
		Domains struct {
			Domain []struct {
				DomainID   string `json:"DomainId"`
				DomainName string `json:"DomainName"`
			} `json:"Domain"`
		} `json:"Domains"`
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := p.request(ctx, params, &result); err != nil {
		return nil, err
	}
	if result.Code != "" {
		return nil, fmt.Errorf("aliyun: %s", result.Message)
	}
	zones := make([]dns.Zone, 0, len(result.Domains.Domain))
	for _, d := range result.Domains.Domain {
		zones = append(zones, dns.Zone{ID: d.DomainID, Domain: d.DomainName})
	}
	return zones, nil
}

func (p *Aliyun) CreateRecord(ctx context.Context, zone dns.Zone, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	rr := hostToSub(input.Name, zone.Domain)
	params := map[string]string{
		"Action":     "AddDomainRecord",
		"DomainName": zone.Domain,
		"RR":         rr,
		"Type":       strings.ToUpper(input.Type),
		"Value":      input.Value,
		"TTL":        fmt.Sprintf("%d", ttl),
	}
	if input.Line != "" {
		params["Line"] = input.Line
	}
	var result struct {
		RecordID string `json:"RecordId"`
		Code     string `json:"Code"`
		Message  string `json:"Message"`
	}
	if err := p.request(ctx, params, &result); err != nil {
		return dns.Record{}, err
	}
	if result.Code != "" {
		return dns.Record{}, fmt.Errorf("aliyun: %s", result.Message)
	}
	return dns.Record{
		RemoteID: result.RecordID,
		Name:     input.Name,
		Type:     strings.ToUpper(input.Type),
		Value:    input.Value,
		TTL:      ttl,
	}, nil
}

func (p *Aliyun) UpdateRecord(ctx context.Context, zone dns.Zone, remoteID string, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	rr := hostToSub(input.Name, zone.Domain)
	params := map[string]string{
		"Action":   "UpdateDomainRecord",
		"RecordId": remoteID,
		"RR":       rr,
		"Type":     strings.ToUpper(input.Type),
		"Value":    input.Value,
		"TTL":      fmt.Sprintf("%d", ttl),
	}
	var result struct {
		RecordID string `json:"RecordId"`
		Code     string `json:"Code"`
		Message  string `json:"Message"`
	}
	if err := p.request(ctx, params, &result); err != nil {
		return dns.Record{}, err
	}
	if result.Code != "" {
		return dns.Record{}, fmt.Errorf("aliyun: %s", result.Message)
	}
	return dns.Record{
		RemoteID: remoteID,
		Name:     input.Name,
		Type:     strings.ToUpper(input.Type),
		Value:    input.Value,
		TTL:      ttl,
	}, nil
}

func (p *Aliyun) DeleteRecord(ctx context.Context, zone dns.Zone, remoteID string) error {
	params := map[string]string{
		"Action":   "DeleteDomainRecord",
		"RecordId": remoteID,
	}
	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := p.request(ctx, params, &result); err != nil {
		return err
	}
	if result.Code != "" {
		return fmt.Errorf("aliyun: %s", result.Message)
	}
	return nil
}

func (p *Aliyun) request(ctx context.Context, params map[string]string, out interface{}) error {
	params["Format"] = "JSON"
	params["Version"] = "2015-01-09"
	params["AccessKeyId"] = p.accessKeyID
	params["SignatureMethod"] = "HMAC-SHA1"
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params["SignatureVersion"] = "1.0"
	params["SignatureNonce"] = uuid.NewString()

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var canonical []string
	for _, k := range keys {
		canonical = append(canonical, percentEncode(k)+"="+percentEncode(params[k]))
	}
	canonicalized := strings.Join(canonical, "&")
	stringToSign := "GET&" + percentEncode("/") + "&" + percentEncode(canonicalized)
	mac := hmac.New(sha1.New, []byte(p.accessKeySecret+"&"))
	_, _ = mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	params["Signature"] = signature

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	endpoint := "https://alidns.aliyuncs.com/?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
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
		return fmt.Errorf("aliyun http %d: %s", resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}

func percentEncode(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(url.QueryEscape(s), "+", "%20"), "*", "%2A"), "%7E", "~")
}
