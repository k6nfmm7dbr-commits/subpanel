package formats

import (
	"strings"
	"subpanel/internal/model"
	"testing"
)

func sampleNodes() []model.Node {
	return []model.Node{
		{Name: "SS-A", Type: "ss", Server: "1.2.3.4", Port: 8388, Cipher: "aes-256-gcm", Password: "pwd", UDP: true},
		{Name: "VM-B", Type: "vmess", Server: "b.example.com", Port: 443, UUID: "u-123", Network: "ws", Host: "b.example.com", Path: "/ws", TLS: true},
		{Name: "TR-C", Type: "trojan", Server: "c.example.com", Port: 443, Password: "tp", SNI: "c.example.com"},
	}
}

func TestAllFormatsGenerate(t *testing.T) {
	nodes := sampleNodes()
	for _, f := range Formats {
		for _, key := range []string{f.ID, f.Path} {
			body, ct, ok := GenerateSubscription(key, nodes, "test")
			if !ok {
				t.Fatalf("format %s (key=%s): not generated", f.ID, key)
			}
			if strings.TrimSpace(body) == "" {
				t.Fatalf("format %s: empty body", f.ID)
			}
			if ct == "" {
				t.Fatalf("format %s: empty content type", f.ID)
			}
		}
	}
}

func TestShareLinks(t *testing.T) {
	for _, n := range sampleNodes() {
		model.NormalizeNode(&n)
		l := model.ShareLink(n)
		if l == "" {
			t.Fatalf("no share link for %s", n.Type)
		}
		switch n.Type {
		case "ss":
			if !strings.HasPrefix(l, "ss://") {
				t.Fatalf("bad ss link: %s", l)
			}
		case "vmess":
			if !strings.HasPrefix(l, "vmess://") {
				t.Fatalf("bad vmess link: %s", l)
			}
		case "trojan":
			if !strings.HasPrefix(l, "trojan://") {
				t.Fatalf("bad trojan link: %s", l)
			}
		}
	}
}

func TestClashContainsAll(t *testing.T) {
	out, _, _ := GenerateSubscription("clash", sampleNodes(), "t")
	for _, n := range sampleNodes() {
		if !strings.Contains(out, n.Name) {
			t.Fatalf("clash missing node %s", n.Name)
		}
	}
	if !strings.Contains(out, "proxy-groups:") || !strings.Contains(out, "rules:") {
		t.Fatal("clash missing groups/rules")
	}
}

func TestBase64Decodable(t *testing.T) {
	out, _, _ := GenerateSubscription("base64", sampleNodes(), "t")
	if !strings.Contains(out, "ss://") && !strings.Contains(out, "vmess://") {
		// base64 body should decode to share links; check raw prefix via helper
		b, err := model.Base64StdEncoding.DecodeString(strings.TrimSpace(out))
		if err != nil {
			t.Fatalf("base64 decode failed: %v", err)
		}
		if !strings.Contains(string(b), "ss://") {
			t.Fatalf("decoded body missing links: %s", string(b))
		}
	}
}
