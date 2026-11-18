# Changelog

## v1.1.0

- 按 sbx 框架重构：`cmd/subpanel` + `internal/{config,model,formats,store,server}` 分层。
- 新增 sbx 风格 CLI：`serve` / `version` / `config-get` / `config-set` / `config-ensure-token` / `secret hex`。
- 配置走 `SUBPANEL_CONF`（默认 `/etc/subpanel/config.json`），支持 `SUBPANEL_LISTEN` / `SUBPANEL_DATA` / `SUBPANEL_ADMIN` 环境变量。
- 默认监听端口改为 **20012**。
- 新增 `/healthz`、优雅退出（SIGINT/SIGTERM）、ReadHeaderTimeout。
- 内嵌面板移至 `internal/server/ui.html`。

## v1.0.0

- 首个版本：Go 单文件二进制 + 内嵌管理面板。
- 11 种订阅格式（Clash/Meta、Surge、Shadowrocket、Loon、QX、Stash、sing-box、v2rayN Base64、分享链接、JSON）。
- 9 种节点协议（ss/ssr/trojan/vmess/vless/hysteria/hysteria2/tuic/wireguard）。
