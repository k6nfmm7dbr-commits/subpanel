# subpanel

自托管的**订阅链接生成面板**：在网页里填一次节点信息，自动生成适配市面上主流代理软件的订阅链接。更新节点后链接不变。

一个 Go 单文件二进制，内嵌管理面板，数据保存在本地 JSON 文件，无数据库依赖。代码结构、配置与 CLI 风格参照 sbx 项目。

## 支持的输出格式（11 种）

| 客户端 | 格式 | 订阅路径 |
|---|---|---|
| Clash / OpenClash | YAML | `/sub/{token}/clash` |
| Clash Meta / Mihomo | YAML（含 hysteria2/tuic 等） | `/sub/{token}/clash-meta` |
| Surge 4/5 | conf | `/sub/{token}/surge` |
| Shadowrocket | Base64 通用订阅 | `/sub/{token}/shadowrocket` |
| Loon | plugin | `/sub/{token}/loon` |
| Quantumult X | conf | `/sub/{token}/qx` |
| Stash | YAML | `/sub/{token}/stash` |
| sing-box | JSON | `/sub/{token}/singbox.json` |
| v2rayN / Xray / NekoRay | Base64 通用订阅 | `/sub/{token}/base64` |
| 分享链接合集 | 纯文本 | `/sub/{token}/links` |
| 原始节点 JSON | JSON | `/sub/{token}/nodes.json` |

支持的节点协议：`ss`、`ssr`、`trojan`、`vmess`、`vless`、`hysteria`、`hysteria2`、`tuic`、`wireguard`。各客户端不支持的协议会自动跳过。

## 快速开始

默认监听端口：**20012**（可通过 `-addr` 或 `SUBPANEL_LISTEN` 修改）。

```sh
# 编译
go build -o subpanel ./cmd/subpanel

# 启动（令牌通过环境变量）
SUBPANEL_ADMIN=your-secret-token ./subpanel serve

# 或显式指定
./subpanel serve -addr :20012 -data nodes.json -admin your-secret-token
```

打开 `http://服务器IP:20012`，输入管理令牌即可使用。

## 命令（sbx 风格）

```sh
subpanel serve                  # 启动面板（默认命令，可省略 serve）
subpanel version                # 显示版本
subpanel config-get listen      # 读取配置 (listen|data|admin_token)
subpanel config-set listen :20012
subpanel config-ensure-token    # 自动生成 admin_token 并写入配置
subpanel secret hex 32          # 输出随机十六进制
```

配置文件默认 `/etc/subpanel/config.json`，由 `SUBPANEL_CONF` 指定。环境变量 `SUBPANEL_LISTEN`、`SUBPANEL_DATA`、`SUBPANEL_ADMIN` 优先级高于配置文件；命令行 `-addr`、`-data`、`-admin` 最高。

管理 API（需 `X-Admin-Token: your-secret-token` 头）：
- `GET /api/sets` — 列出所有配置组
- `POST /api/sets` — 创建/更新配置组
- `DELETE /api/sets/{token}` — 删除配置组
- `GET /api/urls?token={token}` — 列出该组全部订阅链接

订阅链接本身**不需要令牌**，靠随机 token 作唯一标识；token 泄露即等于订阅泄露。

## 代码结构（参照 sbx）

```
cmd/subpanel/main.go        # CLI 子命令、优雅退出
internal/config/            # JSON 配置 + 环境变量覆盖
internal/model/             # 统一节点模型 + 分享链接
internal/formats/           # 11 种客户端订阅格式生成器
internal/store/             # JSON 原子持久化
internal/server/            # HTTP 路由、鉴权、内嵌面板
  ui.html                   # 管理面板（go:embed）
```

## 安全提示

- 务必设置强管理令牌；面板 API 使用明文 HTTP 传输时请勿暴露到公网，建议套 HTTPS 反代。
- 数据文件与配置文件权限为 0600，包含节点密码和令牌。
- 订阅 token 是访问凭证，请妥善保管。

## 许可证

MIT
