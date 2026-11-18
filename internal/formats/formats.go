package formats

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"subpanel/internal/model"
)

// Format describes one output subscription format.
type Format struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"` // /sub/{token}/{id}
	ContentType string `json:"contentType"`
}

var Formats = []Format{
	{"clash", "Clash / Clash Meta", "YAML, works with Clash, Clash.Meta, Mihomo, OpenClash", "clash", "text/yaml; charset=utf-8"},
	{"clash-meta", "Clash Meta (Mihomo)", "YAML with Meta-only fields (hysteria2, tuic)", "clash-meta", "text/yaml; charset=utf-8"},
	{"surge", "Surge", "Surge 4/5 proxy.conf / module", "surge", "text/plain; charset=utf-8"},
	{"shadowrocket", "Shadowrocket", "Base64 universal subscription", "shadowrocket", "text/plain; charset=utf-8"},
	{"loon", "Loon", "Loon plugin", "loon", "text/plain; charset=utf-8"},
	{"quantumultx", "Quantumult X", "Quantumult X subscription", "qx", "text/plain; charset=utf-8"},
	{"stash", "Stash", "Stash YAML profile", "stash", "text/yaml; charset=utf-8"},
	{"singbox", "sing-box", "sing-box JSON outbound array", "singbox.json", "application/json; charset=utf-8"},
	{"base64", "v2rayN / Xray / NekoRay", "Universal Base64 share-link subscription", "base64", "text/plain; charset=utf-8"},
	{"links", "Share links", "Plain text list of all share links", "links", "text/plain; charset=utf-8"},
	{"json", "JSON (all nodes)", "Raw unified node JSON for further processing", "nodes.json", "application/json; charset=utf-8"},
}

func FormatByID(id string) (Format, bool) {
	for _, f := range Formats {
		if f.ID == id {
			return f, true
		}
	}
	return Format{}, false
}

func SupportedType(n model.Node, client string) bool {
	switch client {
	case "clash", "stash":
		return n.Type == "ss" || n.Type == "ssr" || n.Type == "trojan" || n.Type == "vmess" || n.Type == "vless" || n.Type == "hysteria" || n.Type == "hysteria2" || n.Type == "tuic" || n.Type == "wireguard"
	case "clash-meta":
		return SupportedType(n, "clash")
	case "surge":
		return n.Type == "ss" || n.Type == "ssr" || n.Type == "trojan" || n.Type == "vmess" || n.Type == "vless" || n.Type == "hysteria2" || n.Type == "tuic" || n.Type == "wireguard"
	case "loon":
		return n.Type == "ss" || n.Type == "ssr" || n.Type == "trojan" || n.Type == "vmess" || n.Type == "vless" || n.Type == "hysteria2" || n.Type == "tuic"
	case "quantumultx":
		return n.Type == "ss" || n.Type == "ssr" || n.Type == "trojan" || n.Type == "vmess" || n.Type == "vless" || n.Type == "hysteria2"
	case "singbox":
		return n.Type == "ss" || n.Type == "ssr" || n.Type == "trojan" || n.Type == "vmess" || n.Type == "vless" || n.Type == "hysteria" || n.Type == "hysteria2" || n.Type == "tuic" || n.Type == "wireguard"
	}
	return true
}

func yamlQuote(s string) string {
	if s == "" {
		return `""`
	}
	return strconv.Quote(s)
}

func clashProxy(n model.Node, meta bool) string {
	var b strings.Builder
	name := n.Name
	if name == "" {
		name = n.Server
	}
	fmt.Fprintf(&b, "  - name: %s\n    type: %s\n    server: %s\n    port: %d\n", yamlQuote(name), n.Type, yamlQuote(n.Server), n.Port)
	switch n.Type {
	case "ss":
		fmt.Fprintf(&b, "    cipher: %s\n    password: %s\n", yamlQuote(n.Cipher), yamlQuote(n.Password))
	case "ssr":
		fmt.Fprintf(&b, "    cipher: %s\n    password: %s\n    protocol: %s\n    obfs: %s\n", yamlQuote(n.Cipher), yamlQuote(n.Password), yamlQuote(n.SSRProtocol), yamlQuote(n.SSRObfuscation))
		if n.SSRParam != "" {
			fmt.Fprintf(&b, "    obfs-param: %s\n", yamlQuote(n.SSRParam))
		}
	case "trojan":
		fmt.Fprintf(&b, "    password: %s\n", yamlQuote(n.Password))
	case "vmess":
		fmt.Fprintf(&b, "    uuid: %s\n    alterId: %d\n    cipher: %s\n", yamlQuote(n.UUID), n.AlterID, yamlQuote(n.Cipher))
	case "vless":
		fmt.Fprintf(&b, "    uuid: %s\n", yamlQuote(n.UUID))
		if n.UDP {
			b.WriteString("    flow: xtls-rprx-vision\n")
		}
	case "hysteria":
		fmt.Fprintf(&b, "    auth-str: %s\n    up: \"%d Mbps\"\n    down: \"%d Mbps\"\n", yamlQuote(n.Password), n.UpMbps, n.DownMbps)
	case "hysteria2":
		fmt.Fprintf(&b, "    password: %s\n", yamlQuote(n.Password))
		if n.UpMbps > 0 {
			fmt.Fprintf(&b, "    up: \"%d Mbps\"\n", n.UpMbps)
		}
		if n.DownMbps > 0 {
			fmt.Fprintf(&b, "    down: \"%d Mbps\"\n", n.DownMbps)
		}
		if n.Obfs != "" {
			fmt.Fprintf(&b, "    obfs: %s\n    obfs-password: %s\n", yamlQuote(n.Obfs), yamlQuote(n.Password))
		}
	case "tuic":
		fmt.Fprintf(&b, "    uuid: %s\n    password: %s\n", yamlQuote(n.UUID), yamlQuote(n.Password))
	case "wireguard":
		fmt.Fprintf(&b, "    private-key: %s\n    peer-key: %s\n    ip: %s\n    dns: %s\n    mtu: %d\n", yamlQuote(n.PrivateKey), yamlQuote(n.PeerKey), yamlQuote(n.IP), yamlQuote(n.DNS), n.MTU)
	}
	if n.Network != "" && (n.Type == "vmess" || n.Type == "vless") {
		fmt.Fprintf(&b, "    network: %s\n", yamlQuote(n.Network))
		if n.Network == "ws" || n.Network == "http" || n.Network == "h2" {
			if n.Host != "" {
				fmt.Fprintf(&b, "    servername: %s\n    http-opts:\n      headers:\n        Host:\n          - %s\n", yamlQuote(n.Host), yamlQuote(n.Host))
			}
			if n.Path != "" {
				fmt.Fprintf(&b, "    http-opts:\n      path:\n        - %s\n", yamlQuote(n.Path))
			}
		}
		if n.Network == "grpc" && n.Path != "" {
			fmt.Fprintf(&b, "    grpc-opts:\n      grpc-service-name: %s\n", yamlQuote(n.Path))
		}
	}
	if n.TLS || n.Type == "trojan" || n.Type == "hysteria2" || n.Type == "tuic" || n.Type == "vless" {
		b.WriteString("    tls: true\n")
		if n.SNI != "" {
			fmt.Fprintf(&b, "    servername: %s\n", yamlQuote(n.SNI))
		} else if n.Host != "" {
			fmt.Fprintf(&b, "    servername: %s\n", yamlQuote(n.Host))
		}
		if n.SkipCert {
			b.WriteString("    skip-cert-verify: true\n")
		}
		if len(model.ALPNList(n.ALPN)) > 0 && meta {
			fmt.Fprintf(&b, "    alpn:\n")
			for _, a := range model.ALPNList(n.ALPN) {
				fmt.Fprintf(&b, "      - %s\n", yamlQuote(a))
			}
		}
	}
	if n.UDP {
		b.WriteString("    udp: true\n")
	}
	return b.String()
}

func genClash(nodes []model.Node, meta bool) string {
	var b strings.Builder
	b.WriteString("port: 7890\nsocks-port: 7891\nallow-lan: false\nmode: rule\nlog-level: info\nexternal-controller: 127.0.0.1:9090\n\nproxies:\n")
	for _, n := range nodes {
		if SupportedType(n, "clash") {
			b.WriteString(clashProxy(n, meta))
		}
	}
	b.WriteString("\nproxy-groups:\n  - name: PROXY\n    type: select\n    proxies:\n      - DIRECT\n")
	for _, n := range nodes {
		if SupportedType(n, "clash") {
			name := n.Name
			if name == "" {
				name = n.Server
			}
			fmt.Fprintf(&b, "      - %s\n", yamlQuote(name))
		}
	}
	b.WriteString("\nrules:\n  - MATCH,PROXY\n")
	return b.String()
}

func genSurge(nodes []model.Node) string {
	var b strings.Builder
	b.WriteString("[General]\nloglevel = notify\n\n[Proxy]\n")
	for _, n := range nodes {
		if !SupportedType(n, "surge") {
			continue
		}
		name := n.Name
		if name == "" {
			name = n.Server
		}
		switch n.Type {
		case "ss":
			fmt.Fprintf(&b, "%s = ss, %s, %d, encrypt-method=%s, password=%s", name, n.Server, n.Port, n.Cipher, n.Password)
		case "ssr":
			fmt.Fprintf(&b, "%s = ssr, %s, %d, %s, %s, %s, %s", name, n.Server, n.Port, n.Cipher, n.Password, n.SSRProtocol, n.SSRObfuscation)
		case "trojan":
			fmt.Fprintf(&b, "%s = trojan, %s, %d, %s", name, n.Server, n.Port, n.Password)
			if n.SNI != "" {
				fmt.Fprintf(&b, ", sni=%s", n.SNI)
			}
		case "vmess":
			fmt.Fprintf(&b, "%s = vmess, %s, %d, %s", name, n.Server, n.Port, n.UUID)
			if n.TLS {
				b.WriteString(", tls=true")
			}
		case "vless":
			fmt.Fprintf(&b, "%s = vless, %s, %d, %s", name, n.Server, n.Port, n.UUID)
		case "hysteria2":
			fmt.Fprintf(&b, "%s = hysteria2, %s, %d, %s", name, n.Server, n.Port, n.Password)
		case "tuic":
			fmt.Fprintf(&b, "%s = tuic, %s, %d, %s, %s", name, n.Server, n.Port, n.UUID, n.Password)
		case "wireguard":
			fmt.Fprintf(&b, "%s = wireguard, self-ip=%s, private-key=%s, peer-key=%s", name, n.IP, n.PrivateKey, n.PeerKey)
		default:
			continue
		}
		if n.UDP && n.Type != "wireguard" {
			b.WriteString(", udp-relay=true")
		}
		b.WriteString("\n")
	}
	b.WriteString("\n[Proxy Group]\nPROXY = select, DIRECT, ")
	names := []string{}
	for _, n := range nodes {
		if SupportedType(n, "surge") {
			name := n.Name
			if name == "" {
				name = n.Server
			}
			names = append(names, name)
		}
	}
	b.WriteString(strings.Join(names, ", "))
	b.WriteString("\n\n[Rule]\nMATCH,PROXY\n")
	return b.String()
}

func genLoon(nodes []model.Node) string {
	var b strings.Builder
	b.WriteString("[General]\n\n[Proxy]\n")
	for _, n := range nodes {
		if !SupportedType(n, "loon") {
			continue
		}
		name := n.Name
		if name == "" {
			name = n.Server
		}
		switch n.Type {
		case "ss":
			fmt.Fprintf(&b, "%s=ss, %s, %d, %s, %s\n", name, n.Server, n.Port, n.Cipher, n.Password)
		case "ssr":
			fmt.Fprintf(&b, "%s=ssr, %s, %d, %s, %s, %s, %s\n", name, n.Server, n.Port, n.Cipher, n.Password, n.SSRProtocol, n.SSRObfuscation)
		case "trojan":
			fmt.Fprintf(&b, "%s=trojan, %s, %d, %s\n", name, n.Server, n.Port, n.Password)
		case "vmess":
			fmt.Fprintf(&b, "%s=vmess, %s, %d, %s\n", name, n.Server, n.Port, n.UUID)
		case "vless":
			fmt.Fprintf(&b, "%s=vless, %s, %d, %s\n", name, n.Server, n.Port, n.UUID)
		case "hysteria2":
			fmt.Fprintf(&b, "%s=hysteria2, %s, %d, %s\n", name, n.Server, n.Port, n.Password)
		case "tuic":
			fmt.Fprintf(&b, "%s=tuic, %s, %d, %s, %s\n", name, n.Server, n.Port, n.UUID, n.Password)
		}
	}
	return b.String()
}

func genQuantumultX(nodes []model.Node) string {
	var b strings.Builder
	b.WriteString("[General]\n\n[Policy]\n")
	for _, n := range nodes {
		if !SupportedType(n, "quantumultx") {
			continue
		}
		name := n.Name
		if name == "" {
			name = n.Server
		}
		switch n.Type {
		case "ss":
			fmt.Fprintf(&b, "%s=shadowsocks, %s:%d, %s, %s, udp-relay=true\n", name, n.Server, n.Port, n.Cipher, n.Password)
		case "ssr":
			fmt.Fprintf(&b, "%s=shadowsocksr, %s:%d, %s, %s, udp-relay=true\n", name, n.Server, n.Port, n.Cipher, n.Password)
		case "trojan":
			fmt.Fprintf(&b, "%s=trojan, %s:%d, %s, udp-relay=true\n", name, n.Server, n.Port, n.Password)
		case "vmess":
			fmt.Fprintf(&b, "%s=vmess, %s:%d, %s, udp-relay=true\n", name, n.Server, n.Port, n.UUID)
		case "vless":
			fmt.Fprintf(&b, "%s=vless, %s:%d, %s, udp-relay=true\n", name, n.Server, n.Port, n.UUID)
		case "hysteria2":
			fmt.Fprintf(&b, "%s=hysteria2, %s:%d, %s\n", name, n.Server, n.Port, n.Password)
		}
	}
	return b.String()
}

func singboxOutbound(n model.Node) map[string]interface{} {
	name := n.Name
	if name == "" {
		name = n.Server
	}
	out := map[string]interface{}{"tag": name, "type": n.Type, "server": n.Server, "server_port": n.Port}
	switch n.Type {
	case "ss":
		out["method"] = n.Cipher
		out["password"] = n.Password
	case "ssr":
		out["method"] = n.Cipher
		out["password"] = n.Password
		out["protocol"] = n.SSRProtocol
		out["obfs"] = n.SSRObfuscation
	case "trojan":
		out["password"] = n.Password
	case "vmess":
		out["uuid"] = n.UUID
		out["alter_id"] = n.AlterID
		out["security"] = n.Cipher
	case "vless":
		out["uuid"] = n.UUID
	case "hysteria":
		out["auth_str"] = n.Password
		out["up_mbps"] = n.UpMbps
		out["down_mbps"] = n.DownMbps
	case "hysteria2":
		out["password"] = n.Password
	case "tuic":
		out["uuid"] = n.UUID
		out["password"] = n.Password
		out["version"] = n.Version
	case "wireguard":
		out["private_key"] = n.PrivateKey
		out["peer_public_key"] = n.PeerKey
		out["local_address"] = []string{n.IP}
		out["mtu"] = n.MTU
	}
	if n.TLS || n.Type == "trojan" || n.Type == "hysteria2" || n.Type == "tuic" || n.Type == "vless" {
		tls := map[string]interface{}{"enabled": true}
		if n.SNI != "" {
			tls["server_name"] = n.SNI
		}
		if n.SkipCert {
			tls["insecure"] = true
		}
		if len(model.ALPNList(n.ALPN)) > 0 {
			tls["alpn"] = model.ALPNList(n.ALPN)
		}
		out["tls"] = tls
	}
	if n.Network == "ws" || n.Network == "http" || n.Network == "h2" || n.Network == "grpc" {
		out["transport"] = map[string]interface{}{
			"type":         n.Network,
			"host":         n.Host,
			"path":         n.Path,
			"service_name": n.Path,
		}
	}
	return out
}

func genSingBox(nodes []model.Node) string {
	outbounds := []map[string]interface{}{{"type": "selector", "tag": "proxy", "outbounds": []string{}}}
	var names []string
	for _, n := range nodes {
		if SupportedType(n, "singbox") {
			names = append(names, n.Name)
		}
	}
	outbounds[0]["outbounds"] = names
	for _, n := range nodes {
		if SupportedType(n, "singbox") {
			outbounds = append(outbounds, singboxOutbound(n))
		}
	}
	b, _ := json.MarshalIndent(map[string]interface{}{"outbounds": outbounds}, "", "  ")
	return string(b)
}

func genLinks(nodes []model.Node) string {
	var lines []string
	for _, n := range nodes {
		if l := model.ShareLink(n); l != "" {
			lines = append(lines, l)
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}

func GenerateSubscription(formatID string, nodes []model.Node, clientName string) (string, string, bool) {
	// Accept both format ID and subscription path as the lookup key, since
	// URLs use Path (e.g. qx, singbox.json) while the admin API uses ID.
	resolved := formatID
	for _, f := range Formats {
		if formatID == f.ID || formatID == f.Path {
			resolved = f.ID
			break
		}
	}
	switch resolved {
	case "clash":
		return genClash(nodes, false), Formats[0].ContentType, true
	case "clash-meta":
		return genClash(nodes, true), Formats[1].ContentType, true
	case "surge":
		return genSurge(nodes), Formats[2].ContentType, true
	case "shadowrocket", "base64":
		return model.EncodeShareSubscription(nodes), "text/plain; charset=utf-8", true
	case "loon":
		return genLoon(nodes), Formats[4].ContentType, true
	case "quantumultx":
		return genQuantumultX(nodes), Formats[5].ContentType, true
	case "stash":
		return genClash(nodes, true), Formats[6].ContentType, true
	case "singbox":
		return genSingBox(nodes), Formats[7].ContentType, true
	case "links":
		return genLinks(nodes), Formats[9].ContentType, true
	case "json":
		b, _ := json.MarshalIndent(nodes, "", "  ")
		return string(b), Formats[10].ContentType, true
	}
	return "", "", false
}
