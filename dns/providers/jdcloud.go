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
	"sort"
	"strings"
	"time"

	"tudns/dns"
)

func init() {
	dns.Register("jdcloud", func() dns.Provider { return &JDCloud{} })
}

type JDCloud struct {
	accessKey string
	secretKey string
	client    *http.Client
}

func (p *JDCloud) Key() string   { return "jdcloud" }
func (p *JDCloud) Label() string { return "东东云DNS" }

func (p *JDCloud) ConfigFields() []dns.ConfigField {
	return []dns.ConfigField{
		{Name: "access_key", Label: "Access Key", Required: true, Secret: false},
		{Name: "secret_key", Label: "Secret Key", Required: true, Secret: true},
	}
}

func (p *JDCloud) Configure(config map[string]string) error {
	p.accessKey = strings.TrimSpace(config["access_key"])
	p.secretKey = strings.TrimSpace(config["secret_key"])
	if p.accessKey == "" || p.secretKey == "" {
		return fmt.Errorf("access_key and secret_key required")
	}
	p.client = &http.Client{Timeout: 20 * time.Second}
	return nil
}

func (p *JDCloud) Check(ctx context.Context) error {
	_, err := p.ListZones(ctx)
	return err
}

func (p *JDCloud) sign(method, path, query, body string) (string, string) {
	t := time.Now().UTC()
	date := t.Format("20060102T150405Z")
	nonce := fmt.Sprintf("%d", t.UnixNano())
	signedHeaderMap := map[string]string{
		"x-jdcloud-date":  date,
		"x-jdcloud-nonce": nonce,
		"content-type":    "application/json",
		"host":            "dns.jdcloud-api.com",
	}
	keys := make([]string, 0, len(signedHeaderMap))
	for k := range signedHeaderMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var signedHeaders, canHeaders []string
	for _, k := range keys {
		signedHeaders = append(signedHeaders, k)
		canHeaders = append(canHeaders, k+":"+signedHeaderMap[k])
	}

	canonicalReq := strings.Join([]string{
		method,
		path,
		query,
		strings.Join(canHeaders, "\n"),
		"",
		strings.Join(signedHeaders, ";"),
		jdSha256Hex(body),
	}, "\n")

	scope := t.Format("20060102") + "/dns/jdcloud-api"
	stringToSign := strings.Join([]string{
		"JDCLOUD2-HMAC-SHA256",
		date,
		scope,
		jdSha256Hex(canonicalReq),
	}, "\n")

	kDate := jdSignHmac(p.secretKey, t.Format("20060102"))
	kRegion := jdSignHmac(kDate, "dns")
	kService := jdSignHmac(kRegion, "jdcloud-api")
	kSigning := jdSignHmac(kService, "jdcloud2_request")

	mac := hmac.New(sha256.New, []byte(kSigning))
	mac.Write([]byte(stringToSign))
	sig := hex.EncodeToString(mac.Sum(nil))

	auth := fmt.Sprintf("JDCLOUD2-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		p.accessKey, scope, strings.Join(signedHeaders, ";"), sig)

	return date + "|" + nonce, auth
}

func (p *JDCloud) do(ctx context.Context, method, path, query string, body interface{}) (*http.Response, error) {
	var bodyStr string
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyStr = string(b)
		bodyReader = bytes.NewReader(b)
	}
	hdr, auth := p.sign(method, path, query, bodyStr)
	parts := strings.SplitN(hdr, "|", 2)
	req, err := http.NewRequestWithContext(ctx, method, "https://dns.jdcloud-api.com"+path+"?"+query, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", "dns.jdcloud-api.com")
	req.Header.Set("x-jdcloud-date", parts[0])
	req.Header.Set("x-jdcloud-nonce", parts[1])
	req.Header.Set("Authorization", auth)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("jdcloud http %d: %s", resp.StatusCode, string(data))
	}
	return resp, nil
}

type jdCloudZoneResp struct {
	Result struct {
		DataList []struct {
			ID         int    `json:"id"`
			DomainName string `json:"domainName"`
		} `json:"dataList"`
	} `json:"result"`
}

type jdCloudRRResp struct {
	Result struct {
		DataList []struct {
			ID         int    `json:"id"`
			HostRecord string `json:"hostRecord"`
			HostValue  string `json:"hostValue"`
			Type       string `json:"type"`
			TTL        int    `json:"ttl"`
		} `json:"dataList"`
	} `json:"result"`
}

type jdCloudIDResp struct {
	Result struct {
		ID int `json:"id"`
	} `json:"result"`
}

func (p *JDCloud) ListZones(ctx context.Context) ([]dns.Zone, error) {
	resp, err := p.do(ctx, "GET", "/v1/regions/cn-north-1", "domain=yes", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result jdCloudZoneResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	zones := make([]dns.Zone, 0, len(result.Result.DataList))
	for _, z := range result.Result.DataList {
		zones = append(zones, dns.Zone{ID: fmt.Sprintf("%d", z.ID), Domain: z.DomainName})
	}
	return zones, nil
}

func (p *JDCloud) CreateRecord(ctx context.Context, zone dns.Zone, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	body := map[string]interface{}{
		"hostRecord": hostToSub(input.Name, zone.Domain),
		"hostValue":  input.Value,
		"type":       strings.ToUpper(input.Type),
		"ttl":        ttl,
		"viewValue":  "default",
	}
	resp, err := p.do(ctx, "POST", "/v1/regions/cn-north-1/domain/"+zone.ID+"/rrs", "", body)
	if err != nil {
		return dns.Record{}, err
	}
	defer resp.Body.Close()
	var result jdCloudIDResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return dns.Record{}, err
	}
	return dns.Record{RemoteID: fmt.Sprintf("%d", result.Result.ID), Name: input.Name, Type: strings.ToUpper(input.Type), Value: input.Value, TTL: ttl}, nil
}

func (p *JDCloud) UpdateRecord(ctx context.Context, zone dns.Zone, remoteID string, input dns.RecordInput) (dns.Record, error) {
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 600
	}
	body := map[string]interface{}{
		"hostRecord": hostToSub(input.Name, zone.Domain),
		"hostValue":  input.Value,
		"type":       strings.ToUpper(input.Type),
		"ttl":        ttl,
	}
	resp, err := p.do(ctx, "PUT", "/v1/regions/cn-north-1/domain/"+zone.ID+"/rr/"+remoteID, "", body)
	if err != nil {
		return dns.Record{}, err
	}
	defer resp.Body.Close()
	return dns.Record{RemoteID: remoteID, Name: input.Name, Type: strings.ToUpper(input.Type), Value: input.Value, TTL: ttl}, nil
}

func (p *JDCloud) DeleteRecord(ctx context.Context, zone dns.Zone, remoteID string) error {
	resp, err := p.do(ctx, "DELETE", "/v1/regions/cn-north-1/domain/"+zone.ID+"/rr/"+remoteID, "", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func jdSha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func jdSignHmac(key string, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return string(mac.Sum(nil))
}
