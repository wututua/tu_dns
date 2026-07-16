package providers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tudns/dns"
)

func init() {
	dns.Register("huaweicloud", func() dns.Provider { return &HuaweiCloud{} })
}

type HuaweiCloud struct {
	accessKey string
	secretKey string
	region    string
	client    *http.Client
}

func (p *HuaweiCloud) Key() string   { return "huaweicloud" }
func (p *HuaweiCloud) Label() string { return "为为云DNS" }

func (p *HuaweiCloud) ConfigFields() []dns.ConfigField {
	return []dns.ConfigField{
		{Name: "access_key", Label: "Access Key (AK)", Required: true, Secret: false},
		{Name: "secret_key", Label: "Secret Access Key (SK)", Required: true, Secret: true},
		{Name: "region", Label: "区域", Required: false, Secret: false, Description: "默认 cn-east-2"},
	}
}

func (p *HuaweiCloud) Configure(config map[string]string) error {
	p.accessKey = strings.TrimSpace(config["access_key"])
	p.secretKey = strings.TrimSpace(config["secret_key"])
	p.region = strings.TrimSpace(config["region"])
	if p.accessKey == "" || p.secretKey == "" {
		return fmt.Errorf("access_key and secret_key required")
	}
	if p.region == "" {
		p.region = "cn-east-2"
	}
	p.client = &http.Client{Timeout: 20 * time.Second}
	return nil
}

func (p *HuaweiCloud) Check(ctx context.Context) error {
	_, err := p.ListZones(ctx)
	return err
}

func (p *HuaweiCloud) endpoint() string {
	return fmt.Sprintf("https://dns.%s.myhuaweicloud.com/v2", p.region)
}

func (p *HuaweiCloud) sign(req *http.Request) {
	t := time.Now().UTC()
	req.Header.Set("X-Sdk-Date", t.Format("20060102T150405Z"))
	signedHeaders := "host;x-sdk-date"
	host := req.URL.Host

	canonicalReq := strings.Join([]string{
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		"host:" + host,
		"x-sdk-date:" + req.Header.Get("X-Sdk-Date"),
		"",
		signedHeaders,
		hex.EncodeToString(hwEmptyHash()),
	}, "\n")

	scope := t.Format("20060102") + "/" + p.region + "/dns"
	stringToSign := strings.Join([]string{
		"HWS-SDK-SHA256",
		req.Header.Get("X-Sdk-Date"),
		scope,
		hex.EncodeToString(hwHashString(canonicalReq)),
	}, "\n")

	signingKey := p.deriveKey(t)
	mac := hmac.New(sha256.New, signingKey)
	mac.Write([]byte(stringToSign))
	sig := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("Authorization", fmt.Sprintf(
		"HWS-SDK-HMAC-SHA256 Access=%s, SignedHeaders=%s, Signature=%s",
		p.accessKey, signedHeaders, sig,
	))
}

func (p *HuaweiCloud) deriveKey(t time.Time) []byte {
	date := t.Format("20060102")
	h1 := hwHmac([]byte("HWS"+p.secretKey), date)
	h2 := hwHmac(h1, p.region)
	h3 := hwHmac(h2, "dns")
	return h3
}

func (p *HuaweiCloud) do(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.endpoint()+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	p.sign(req)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("huaweicloud http %d: %s", resp.StatusCode, string(data))
	}
	return resp, nil
}

func (p *HuaweiCloud) ListZones(ctx context.Context) ([]dns.Zone, error) {
	resp, err := p.do(ctx, "GET", "/zones?limit=100", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Zones []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"zones"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	zones := make([]dns.Zone, 0, len(result.Zones))
	for _, z := range result.Zones {
		zones = append(zones, dns.Zone{ID: z.ID, Domain: strings.TrimSuffix(z.Name, ".")})
	}
	return zones, nil
}

func (p *HuaweiCloud) CreateRecord(ctx context.Context, zone dns.Zone, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	body := map[string]interface{}{
		"name":    hostToSub(input.Name, zone.Domain) + "." + zone.Domain + ".",
		"type":    strings.ToUpper(input.Type),
		"records": []string{input.Value},
		"ttl":     ttl,
	}
	resp, err := p.do(ctx, "POST", "/zones/"+zone.ID+"/recordsets", body)
	if err != nil {
		return dns.Record{}, err
	}
	defer resp.Body.Close()
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return dns.Record{}, err
	}
	return dns.Record{RemoteID: result.ID, Name: input.Name, Type: strings.ToUpper(input.Type), Value: input.Value, TTL: ttl}, nil
}

func (p *HuaweiCloud) UpdateRecord(ctx context.Context, zone dns.Zone, remoteID string, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	body := map[string]interface{}{
		"name":    hostToSub(input.Name, zone.Domain) + "." + zone.Domain + ".",
		"type":    strings.ToUpper(input.Type),
		"records": []string{input.Value},
		"ttl":     ttl,
	}
	resp, err := p.do(ctx, "PUT", "/zones/"+zone.ID+"/recordsets/"+remoteID, body)
	if err != nil {
		return dns.Record{}, err
	}
	defer resp.Body.Close()
	return dns.Record{RemoteID: remoteID, Name: input.Name, Type: strings.ToUpper(input.Type), Value: input.Value, TTL: ttl}, nil
}

func (p *HuaweiCloud) DeleteRecord(ctx context.Context, zone dns.Zone, remoteID string) error {
	resp, err := p.do(ctx, "DELETE", "/zones/"+zone.ID+"/recordsets/"+remoteID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func hwEmptyHash() []byte {
	h := sha256.Sum256(nil)
	return h[:]
}

func hwHashString(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

func hwHmac(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
