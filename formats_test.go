package main

import (
	"strings"
	"testing"
)

func sampleNodes() []Node {
	return []Node{
		{Name: "SS-A", Type: "ss", Server: "1.2.3.4", Port: 8388, Cipher: "aes-256-gcm", Password: "pwd", UDP: true},
		{Name: "VM-B", Type: "vmess", Server: "b.example.com", Port: 443, UUID: "u-123", Network: "ws", Host: "b.example.com", Path: "/ws", TLS: true},
		{Name: "TR-C", Type: "trojan", Server: "c.example.com", Port: 443, Password: "tp", SNI: "c.example.com"},
	}
}

func TestAllFormatsGenerate(t *testing.T) {
	nodes := sampleNodes()
	for _, f := range formats {
		for _, key := range []string{f.ID, f.Path} {
			body, ct, ok := generateSubscription(key, nodes, "test")
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
		normalizeNode(&n)
		l := shareLink(n)
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
	out, _, _ := generateSubscription("clash", sampleNodes(), "t")
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
	out, _, _ := generateSubscription("base64", sampleNodes(), "t")
	if !strings.Contains(out, "ss://") && !strings.Contains(out, "vmess://") {
		// base64 body should decode to share links; check raw prefix via helper
		b, err := base64StdEncoding.DecodeString(strings.TrimSpace(out))
		if err != nil {
			t.Fatalf("base64 decode failed: %v", err)
		}
		if !strings.Contains(string(b), "ss://") {
			t.Fatalf("decoded body missing links: %s", string(b))
		}
	}
}

func TestStoreRoundTrip(t *testing.T) {
	path := t.TempDir() + "/nodes.json"
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	set := &NodeSet{Token: "tok1", Name: "g", Nodes: sampleNodes()}
	if err := s.Upsert(set); err != nil {
		t.Fatal(err)
	}
	s2, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s2.Get("tok1")
	if !ok || len(got.Nodes) != 3 {
		t.Fatalf("round trip failed: %+v", got)
	}
}
