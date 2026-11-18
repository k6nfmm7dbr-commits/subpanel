# subpanel

自托管订阅面板：网页里填一次节点，生成主流客户端可用的订阅链接。改节点不用改链接。

Go 单文件二进制，面板内嵌，数据存本地 JSON，不需要数据库。

## 安装

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/k6nfmm7dbr-commits/subpanel/main/install.sh)
```

需要 root。装完打开 `http://服务器IP:20012`，管理令牌在安装结束时打印，也可以 `subpanel config-get admin_token` 查。

```bash
# 换端口
bash <(curl -fsSL https://raw.githubusercontent.com/k6nfmm7dbr-commits/subpanel/main/install.sh) --port 8080
```

Debian 11+ / Ubuntu 20.04+ / RHEL 8+ / Fedora 38+ / Alpine 3.18+，x86_64、arm64 等 7 种架构，systemd 与 OpenRC 都支持。脚本会下载二进制并校验 SHA256，装完自检 `/healthz`。

公网访问要放行对应端口。面板走明文 HTTP，长期开在公网请套一层 HTTPS 反代。

## 订阅格式

| 客户端 | 格式 | 路径 |
|---|---|---|
| Clash / OpenClash | YAML | `/sub/{token}/clash` |
| Clash Meta / Mihomo | YAML | `/sub/{token}/clash-meta` |
| Surge 4/5 | conf | `/sub/{token}/surge` |
| Shadowrocket | Base64 | `/sub/{token}/shadowrocket` |
| Loon | plugin | `/sub/{token}/loon` |
| Quantumult X | conf | `/sub/{token}/qx` |
| Stash | YAML | `/sub/{token}/stash` |
| sing-box | JSON | `/sub/{token}/singbox.json` |
| v2rayN / Xray | Base64 | `/sub/{token}/base64` |
| 分享链接 | 纯文本 | `/sub/{token}/links` |
| 原始节点 | JSON | `/sub/{token}/nodes.json` |

支持 ss、ssr、trojan、vmess、vless、hysteria、hysteria2、tuic、wireguard，客户端不支持的协议自动跳过。订阅链接不需要令牌，靠随机 token 区分，token 泄露等于订阅泄露。

## 命令

```bash
subpanel serve                  # 启动（默认命令）
subpanel version                # 版本
subpanel config-get listen      # 读配置 (listen|data|admin_token)
subpanel config-set listen :20012
subpanel config-ensure-token    # 生成并写入管理令牌
subpanel secret hex 32          # 随机十六进制
```

配置文件 `/etc/subpanel/config.json`。环境变量 `SUBPANEL_LISTEN`、`SUBPANEL_DATA`、`SUBPANEL_ADMIN` 覆盖文件，命令行 `-addr`、`-data`、`-admin` 优先级最高。

管理 API 需要 `X-Admin-Token` 头：`GET|POST /api/sets`、`DELETE /api/sets/{token}`、`GET /api/urls?token={token}`。

## 文件

```text
/usr/local/bin/subpanel      二进制
/etc/subpanel/config.json    配置（0600）
/var/lib/subpanel/nodes.json 节点数据（0600）
```

服务名 `subpanel`，以系统用户 `subpanel` 运行。

## 升级与卸载

重跑安装命令就是升级，配置和令牌保留。

```bash
# 卸载，保留 /etc/subpanel 与数据
bash <(curl -fsSL https://raw.githubusercontent.com/k6nfmm7dbr-commits/subpanel/main/install.sh) --uninstall

# 连配置和数据一起删
bash <(curl -fsSL https://raw.githubusercontent.com/k6nfmm7dbr-commits/subpanel/main/install.sh) --uninstall --purge
```

## 从源码编译

需要 Go 1.23 以上。

```bash
go build -o subpanel ./cmd/subpanel
```

代码分层：`cmd/subpanel` 入口，`internal/{config,model,formats,store,server}`，面板在 `internal/server/ui.html`（go:embed）。

## 许可证

MIT
