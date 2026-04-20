package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	C "github.com/metacubex/mihomo/constant"
)

func TestParseProxiesGlobalIncludesExternalProviderNodes(t *testing.T) {
	previousHomeDir := C.Path.HomeDir()
	t.Cleanup(func() {
		C.SetHomeDir(previousHomeDir)
	})

	homeDir := t.TempDir()
	C.SetHomeDir(homeDir)

	providerPath := filepath.Join(homeDir, "providers.yaml")
	providerContent := `proxies:
  - name: Provider Node A
    type: ss
    server: 127.0.0.1
    port: 8388
    cipher: aes-128-gcm
    password: provider-a
  - name: Provider Node B
    type: ss
    server: 127.0.0.1
    port: 8389
    cipher: aes-128-gcm
    password: provider-b
`
	if err := os.WriteFile(providerPath, []byte(providerContent), 0o600); err != nil {
		t.Fatalf("write provider file: %v", err)
	}

	proxies, providers, err := parseProxies(&RawConfig{
		ProxyProvider: map[string]map[string]any{
			"desktop-provider": {
				"type": "file",
				"path": providerPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("parse proxies: %v", err)
	}

	for name, provider := range providers {
		if err := provider.Initial(); err != nil {
			t.Fatalf("initialize provider %s: %v", name, err)
		}
	}

	globalProxy, ok := proxies["GLOBAL"]
	if !ok {
		t.Fatal("GLOBAL proxy group was not created")
	}

	var payload struct {
		All []string `json:"all"`
	}
	data, err := globalProxy.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal GLOBAL proxy: %v", err)
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode GLOBAL proxy payload: %v", err)
	}

	for _, expected := range []string{"Provider Node A", "Provider Node B"} {
		if !containsProxyName(payload.All, expected) {
			t.Fatalf("GLOBAL group missing provider proxy %q, got %v", expected, payload.All)
		}
	}
}

func TestParseProxiesSupportsAnyTLSNodes(t *testing.T) {
	proxies, _, err := parseProxies(&RawConfig{
		Proxy: []map[string]any{
			{
				"name":                 "AnyTLS Node",
				"type":                 "anytls",
				"server":               "example.com",
				"port":                 443,
				"password":             "secret",
				"udp":                  true,
				"skip-cert-verify":     true,
				"client-fingerprint":   "chrome",
				"idle-session-timeout": 30,
				"min-idle-session":     1,
				"interface-name":       "",
				"routing-mark":         0,
			},
		},
	})
	if err != nil {
		t.Fatalf("parse proxies: %v", err)
	}

	proxy, ok := proxies["AnyTLS Node"]
	if !ok {
		t.Fatal("AnyTLS proxy was not created")
	}
	if proxy.Type() != C.AnyTLS {
		t.Fatalf("unexpected proxy type: got %s", proxy.Type())
	}

	globalProxy, ok := proxies["GLOBAL"]
	if !ok {
		t.Fatal("GLOBAL proxy group was not created")
	}

	var payload struct {
		All []string `json:"all"`
	}
	data, err := globalProxy.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal GLOBAL proxy: %v", err)
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode GLOBAL proxy payload: %v", err)
	}
	if !containsProxyName(payload.All, "AnyTLS Node") {
		t.Fatalf("GLOBAL group missing AnyTLS node, got %v", payload.All)
	}
}

func containsProxyName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
