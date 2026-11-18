# subpanel

自托管的**订阅链接生成面板**：在网页里填一次节点信息，自动生成适配市面上主流代理软件的订阅链接。更新节点后链接不变。

一个 Go 单文件二进制，内嵌管理面板，数据保存在本地 JSON 文件，无数据库依赖。

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

```sh
# 下载/编译后运行
SUBPANEL_ADMIN=your-secret-token ./subpanel -addr :8080 -data nodes.json

# 或用参数
./subpanel -admin your-secret-token -addr 127.0.0.1:8080
```

打开 `http://服务器IP:8080`，输入管理令牌即可使用。

管理 API（需 `X-Admin-Token: your-secret-token` 头）：
- `GET /api/sets` — 列出所有配置组
- `POST /api/sets` — 创建/更新配置组
- `DELETE /api/sets/{token}` — 删除配置组
- `GET /api/urls?token={token}` — 列出该组全部订阅链接

订阅链接本身**不需要令牌**，靠随机 token 作唯一标识；token 泄露即等于订阅泄露。

## 从源码编译

```sh
go build -o subpanel .
```

需要 Go 1.23+。

## 架构

- `model.go` — 统一节点模型 + 分享链接（`ss://` `vmess://` 等）生成
- `formats.go` — 11 种客户端订阅格式生成器
- `store.go` — JSON 文件持久化（原子写入）
- `server.go` — HTTP 路由与鉴权
- `ui.html` — 内嵌管理面板（go:embed）

## 安全提示

- 务必设置强管理令牌；面板 API 使用明文 HTTP 传输时请勿暴露到公网，建议套 HTTPS 反代。
- 数据文件 `nodes.json` 含节点密码，权限为 0600。
- 订阅 token 是访问凭证，请妥善保管。

## 许可证

MIT
