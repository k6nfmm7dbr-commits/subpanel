package model

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Node is the unified node representation entered in the web panel.
// Only ss / ssr / trojan / vmess / vless / hysteria / hysteria2 / tuic /
// wireguard are supported by all major clients; unsupported combos are
// simply omitted from a given client's output.
type Node struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // ss, ssr, trojan, vmess, vless, hysteria, hysteria2, tuic, wireguard
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	UUID     string `json:"uuid,omitempty"`
	Cipher   string `json:"cipher,omitempty"`  // ss method / vmess alterId cipher / wireguard
	Network  string `json:"network,omitempty"` // tcp, ws, grpc, h2, http, quic, kcp
	Host     string `json:"host,omitempty"`    // ws/h2 Host or HTTP Host
	Path     string `json:"path,omitempty"`    // ws/h2 path
	TLS      bool   `json:"tls,omitempty"`
	SNI      string `json:"sni,omitempty"`
	ALPN     string `json:"alpn,omitempty"` // comma separated
	SkipCert bool   `json:"skipCert,omitempty"`
	// SSR specific
	SSRProtocol    string `json:"ssrProtocol,omitempty"`
	SSRObfuscation string `json:"ssrObfuscation,omitempty"`
	SSRParam       string `json:"ssrParam,omitempty"`
	// VMess specific
	AlterID int `json:"alterId,omitempty"`
	// Hysteria / TUIC
	UpMbps   int    `json:"upMbps,omitempty"`
	DownMbps int    `json:"downMbps,omitempty"`
	Obfs     string `json:"obfs,omitempty"`    // hysteria2 salamander
	Version  string `json:"version,omitempty"` // tuic version
	// WireGuard
	PrivateKey string `json:"privateKey,omitempty"`
	PeerKey    string `json:"peerKey,omitempty"`
	IP         string `json:"ip,omitempty"`
	DNS        string `json:"dns,omitempty"`
	MTU        int    `json:"mtu,omitempty"`
	// misc
	Group    string `json:"group,omitempty"`
	UDP      bool   `json:"udp,omitempty"`
	ClientID string `json:"clientId,omitempty"` // subscription token for this node set
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func NormalizeNode(n *Node) {
	if n.ID == "" {
		n.ID = newID()
	}
	n.Type = strings.ToLower(strings.TrimSpace(n.Type))
	n.Name = strings.TrimSpace(n.Name)
	n.Server = strings.TrimSpace(n.Server)
	if n.Port == 0 {
		switch n.Type {
		case "hysteria", "hysteria2":
			n.Port = 443
		case "tuic":
			n.Port = 443
		default:
			n.Port = 8388
		}
	}
	if n.Cipher == "" {
		switch n.Type {
		case "ss":
			n.Cipher = "aes-256-gcm"
		case "vmess":
			n.Cipher = "auto"
		case "ssr":
			n.Cipher = "aes-256-cfb"
		case "wireguard":
			n.Cipher = "chacha20-poly1305"
		}
	}
	if n.Network == "" {
		n.Network = "tcp"
	}
	if n.Network == "websocket" {
		n.Network = "ws"
	}
	if n.AlterID == 0 && n.Type == "vmess" {
		n.AlterID = 0
	}
	if n.MTU == 0 && n.Type == "wireguard" {
		n.MTU = 1420
	}
	if n.Version == "" && n.Type == "tuic" {
		n.Version = "5"
	}
}

func ALPNList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func b64url(s string) string {
	return strings.TrimRight(Base64URLEncoding.EncodeToString([]byte(s)), "=")
}

// ---------- Share links (ss:// ssr:// vmess:// vless:// trojan:// …) ----------

func ShareLink(n Node) string {
	switch n.Type {
	case "ss":
		// SIP002: ss://base64(method:password)@server:port#name
		userinfo := b64url(n.Cipher + ":" + n.Password)
		u := &url.URL{Scheme: "ss", User: url.User(userinfo), Host: fmt.Sprintf("%s:%d", n.Server, n.Port), Fragment: n.Name}
		return u.String()
	case "ssr":
		// ssr://base64(server:port:protocol:method:obfs:base64(password)/?obfsparam=...&protoparam=...&remarks=...)
		main := fmt.Sprintf("%s:%d:%s:%s:%s:%s", n.Server, n.Port, n.SSRProtocol, n.Cipher, n.SSRObfuscation, b64url(n.Password))
		var q []string
		if n.SSRParam != "" {
			q = append(q, "obfsparam="+b64url(n.SSRParam))
		}
		if n.Name != "" {
			q = append(q, "remarks="+b64url(n.Name))
		}
		if len(q) > 0 {
			main += "/?" + strings.Join(q, "&")
		}
		return "ssr://" + b64url(main)
	case "vmess":
		obj := map[string]interface{}{
			"v": "2", "ps": n.Name, "add": n.Server, "port": strconv.Itoa(n.Port),
			"id": n.UUID, "aid": strconv.Itoa(n.AlterID), "net": n.Network,
			"type": "none", "host": n.Host, "path": n.Path, "tls": "tls",
		}
		if !n.TLS {
			obj["tls"] = ""
		}
		if n.SNI != "" {
			obj["sni"] = n.SNI
		}
		b, _ := json.Marshal(obj)
		return "vmess://" + Base64StdEncoding.EncodeToString(b)
	case "vless":
		q := url.Values{}
		if n.Network != "tcp" {
			q.Set("type", n.Network)
		} else {
			q.Set("type", "tcp")
		}
		if n.TLS {
			q.Set("security", "tls")
			if n.SNI != "" {
				q.Set("sni", n.SNI)
			}
			if len(ALPNList(n.ALPN)) > 0 {
				q.Set("alpn", n.ALPN)
			}
		} else {
			q.Set("security", "none")
		}
		if n.Host != "" {
			q.Set("host", n.Host)
		}
		if n.Path != "" {
			q.Set("path", n.Path)
		}
		if n.UDP {
			q.Set("flow", "xtls-rprx-vision")
		}
		u := &url.URL{Scheme: "vless", User: url.User(n.UUID), Host: fmt.Sprintf("%s:%d", n.Server, n.Port), RawQuery: q.Encode(), Fragment: n.Name}
		return u.String()
	case "trojan":
		q := url.Values{}
		if n.SNI != "" {
			q.Set("sni", n.SNI)
		}
		if n.ALPN != "" {
			q.Set("alpn", n.ALPN)
		}
		if n.SkipCert {
			q.Set("allowInsecure", "1")
		}
		u := &url.URL{Scheme: "trojan", User: url.User(n.Password), Host: fmt.Sprintf("%s:%d", n.Server, n.Port), RawQuery: q.Encode(), Fragment: n.Name}
		return u.String()
	case "hysteria":
		q := url.Values{}
		q.Set("up", fmt.Sprintf("%d", n.UpMbps))
		q.Set("down", fmt.Sprintf("%d", n.DownMbps))
		if n.Obfs != "" {
			q.Set("obfs", n.Obfs)
		}
		if n.SNI != "" {
			q.Set("peer", n.SNI)
		}
		if n.SkipCert {
			q.Set("insecure", "1")
		}
		if n.ALPN != "" {
			q.Set("alpn", n.ALPN)
		}
		u := &url.URL{Scheme: "hysteria", User: url.User(n.Password), Host: fmt.Sprintf("%s:%d", n.Server, n.Port), RawQuery: q.Encode(), Fragment: n.Name}
		return u.String()
	case "hysteria2", "hy2":
		q := url.Values{}
		q.Set("up", fmt.Sprintf("%d", n.UpMbps))
		q.Set("down", fmt.Sprintf("%d", n.DownMbps))
		if n.Obfs != "" {
			q.Set("obfs", n.Obfs)
		}
		if n.SNI != "" {
			q.Set("sni", n.SNI)
		}
		if n.SkipCert {
			q.Set("insecure", "1")
		}
		u := &url.URL{Scheme: "hysteria2", User: url.User(n.Password), Host: fmt.Sprintf("%s:%d", n.Server, n.Port), RawQuery: q.Encode(), Fragment: n.Name}
		return u.String()
	case "tuic":
		q := url.Values{}
		q.Set("version", n.Version)
		q.Set("alpn", n.ALPN)
		if n.SNI != "" {
			q.Set("sni", n.SNI)
		}
		if n.SkipCert {
			q.Set("allow_insecure", "1")
		}
		if n.UDP {
			q.Set("udp", "1")
		}
		user := n.UUID + ":" + n.Password
		u := &url.URL{Scheme: "tuic", User: url.User(user), Host: fmt.Sprintf("%s:%d", n.Server, n.Port), RawQuery: q.Encode(), Fragment: n.Name}
		return u.String()
	case "wireguard":
		// WireGuard does not have a universal share scheme; emit a standard
		// WG config block as a data URI friendly text in share collection.
		return fmt.Sprintf("[Interface]\nPrivateKey = %s\nAddress = %s\nDNS = %s\nMTU = %d\n\n[Peer]\nPublicKey = %s\nEndpoint = %s:%d\nAllowedIPs = 0.0.0.0/0, ::/0\n", n.PrivateKey, n.IP, n.DNS, n.MTU, n.PeerKey, n.Server, n.Port)
	}
	return ""
}

func isWireGuard(n Node) bool { return n.Type == "wireguard" }

// EncodeShareSubscription produces the universal Base64 subscription body
// used by v2rayN / Xray / Shadowrocket / NekoRay / singular apps.
func EncodeShareSubscription(nodes []Node) string {
	var lines []string
	for _, n := range nodes {
		if isWireGuard(n) {
			continue
		}
		if l := ShareLink(n); l != "" {
			lines = append(lines, l)
		}
	}
	return Base64StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))
}
