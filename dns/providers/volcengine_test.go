package providers

import (
	"testing"

	"tudns/dns"
)

func TestVolcengineConfigure(t *testing.T) {
	t.Parallel()
	p := &Volcengine{}
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

func TestVolcengineMetadata(t *testing.T) {
	t.Parallel()
	p, ok := dns.New("volcengine")
	if !ok || p.Key() != "volcengine" || p.Label() == "" {
		t.Fatalf("provider not registered: %#v, %v", p, ok)
	}
}
