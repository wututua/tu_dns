package providers

import (
	"testing"

	"tudns/dns"
)

func TestBaiduCloudConfigure(t *testing.T) {
	t.Parallel()
	p := &BaiduCloud{}
	if err := p.Configure(map[string]string{}); err == nil {
		t.Fatal("expected missing credentials error")
	}
	if err := p.Configure(map[string]string{"access_key": "ak", "secret_key": "sk"}); err != nil {
		t.Fatalf("configure: %v", err)
	}
	if p.client == nil {
		t.Fatal("client was not initialized")
	}
}

func TestBaiduCloudMetadata(t *testing.T) {
	t.Parallel()
	p, ok := dns.New("baidu")
	if !ok || p.Key() != "baidu" || p.Label() == "" {
		t.Fatalf("provider not registered: %#v, %v", p, ok)
	}
	if got := normalizedTTL(0); got != 600 {
		t.Fatalf("normalizedTTL(0) = %d", got)
	}
}
