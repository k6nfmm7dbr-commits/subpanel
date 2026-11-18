#!/usr/bin/env bash
# subpanel 一键安装 / 升级 / 卸载
#
#   bash <(curl -fsSL https://raw.githubusercontent.com/k6nfmm7dbr-commits/subpanel/main/install.sh)
#   ... --upgrade      重新安装最新二进制，保留配置与令牌
#   ... --uninstall    卸载（保留数据）
#   ... --uninstall --purge  连配置和数据一起删
#   ... --port 8080    指定监听端口
#   ... --listen 127.0.0.1:20012   完整指定监听地址
set -euo pipefail

REPO="k6nfmm7dbr-commits/subpanel"
DIST_REF="dist"
SERVICE="subpanel"
DEFAULT_PORT=20012

PREFIX="${SUBPANEL_PREFIX:-/usr/local}"
CONF_DIR="${SUBPANEL_ETC:-/etc/subpanel}"
DATA_DIR="${SUBPANEL_VAR:-/var/lib/subpanel}"
BIN="$PREFIX/bin/subpanel"
CONF="$CONF_DIR/config.json"

ACTION=install
PORT=""
LISTEN=""
PURGE=0
NO_SERVICE="${SUBPANEL_NO_SERVICE:-0}"

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  C_OK=$'\033[32m'; C_ERR=$'\033[31m'; C_DIM=$'\033[2m'; C_B=$'\033[1m'; C_END=$'\033[0m'
else
  C_OK=; C_ERR=; C_DIM=; C_B=; C_END=
fi

say()  { printf '%s\n' "$*"; }
ok()   { printf '%s✓%s %s\n' "$C_OK" "$C_END" "$*"; }
warn() { printf '%s!%s %s\n' "$C_ERR" "$C_END" "$*" >&2; }
die()  { printf '%s✗%s %s\n' "$C_ERR" "$C_END" "$*" >&2; exit 1; }
dim()  { printf '%s%s%s\n' "$C_DIM" "$*" "$C_END"; }

usage() {
  cat <<'EOF'
subpanel 安装脚本

用法:
  bash install.sh                安装（已安装则升级，保留配置与令牌）
  bash install.sh --upgrade      同上，显式升级
  bash install.sh --uninstall    卸载，保留 /etc/subpanel 与数据
  bash install.sh --uninstall --purge   连配置和数据一起删

选项:
  --port N          监听端口，默认 20012
  --listen ADDR     完整监听地址，如 127.0.0.1:20012（覆盖 --port）
  --no-service      只装二进制和配置，不装系统服务
  -h, --help        显示本页
EOF
}

parse_args() {
  while [ $# -gt 0 ]; do
    case "$1" in
      install|--install) ACTION=install ;;
      --upgrade) ACTION=install ;;
      --uninstall|uninstall) ACTION=uninstall ;;
      --purge) PURGE=1 ;;
      --no-service) NO_SERVICE=1 ;;
      --port)
        [ $# -ge 2 ] || die "--port 需要参数"
        PORT="$2"; shift ;;
      --port=*) PORT="${1#*=}" ;;
      --listen)
        [ $# -ge 2 ] || die "--listen 需要参数"
        LISTEN="$2"; shift ;;
      --listen=*) LISTEN="${1#*=}" ;;
      -h|--help|help) usage; exit 0 ;;
      *) die "未知参数: $1（--help 查看用法）" ;;
    esac
    shift
  done
  if [ -n "$PORT" ] && ! printf '%s' "$PORT" | grep -Eq '^[0-9]+$'; then
    die "端口必须是数字: $PORT"
  fi
  if [ -n "$PORT" ] && { [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; }; then
    die "端口超出范围: $PORT"
  fi
}

need_root() {
  [ "$(id -u)" = 0 ] || die "需要 root，请用 sudo 执行"
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)        echo amd64 ;;
    aarch64|arm64)       echo arm64 ;;
    armv7l|armv7)        echo armv7 ;;
    armv6l|armv6)        echo armv6 ;;
    i386|i486|i586|i686) echo 386 ;;
    s390x)               echo s390x ;;
    riscv64)             echo riscv64 ;;
    *) return 1 ;;
  esac
}

detect_init() {
  if [ "$NO_SERVICE" = 1 ]; then echo none
  elif [ -d /run/systemd/system ] && command -v systemctl >/dev/null 2>&1; then echo systemd
  elif command -v rc-update >/dev/null 2>&1 && [ -d /etc/init.d ]; then echo openrc
  else echo none
  fi
}

fetch() {
  local url="$1" out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 2 --connect-timeout 10 --max-time 300 -o "$out" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -T 20 -t 2 -O "$out" "$url"
  else
    die "缺少 curl 或 wget"
  fi
}

base_urls() {
  if [ -n "${SUBPANEL_BASE_URL:-}" ]; then
    printf '%s\n' "$SUBPANEL_BASE_URL"
    return
  fi
  printf '%s\n' "https://raw.githubusercontent.com/$REPO/$DIST_REF"
  # raw 在国内常被阻断，失败时退到 jsDelivr 镜像
  printf '%s\n' "https://cdn.jsdelivr.net/gh/$REPO@$DIST_REF"
}

download_binary() {
  local dir="$1" arch="$2" url got=0
  while IFS= read -r url; do
    [ -n "$url" ] || continue
    if fetch "$url/SHA256SUMS" "$dir/SHA256SUMS" 2>/dev/null; then
      if fetch "$url/subpanel-linux-$arch" "$dir/subpanel-linux-$arch" 2>/dev/null; then
        got=1
        break
      fi
    fi
    dim "  源不可用，换下一个: $url"
  done < <(base_urls)
  [ "$got" = 1 ] || die "下载失败，请检查网络或代理设置"

  local line
  line=$(grep " subpanel-linux-$arch\$" "$dir/SHA256SUMS" || true)
  [ -n "$line" ] || die "SHA256SUMS 里没有 linux/$arch 的条目"
  printf '%s\n' "$line" > "$dir/check.txt"
  ( cd "$dir" && sha256sum -c check.txt >/dev/null ) || die "校验失败，文件可能损坏或被篡改"
  ok "下载并校验通过（linux/$arch）"
}

ensure_user() {
  id subpanel >/dev/null 2>&1 && return 0
  if command -v useradd >/dev/null 2>&1; then
    useradd --system --no-create-home --home-dir "$DATA_DIR" \
            --shell /usr/sbin/nologin subpanel 2>/dev/null && return 0
    useradd -r -M -d "$DATA_DIR" -s /sbin/nologin subpanel 2>/dev/null && return 0
  elif command -v adduser >/dev/null 2>&1; then
    adduser -S -D -H -h "$DATA_DIR" -s /sbin/nologin subpanel 2>/dev/null && return 0
  fi
  return 1
}

resolved_listen() {
  if [ -n "$LISTEN" ]; then
    printf '%s' "$LISTEN"; return
  fi
  printf ':%s' "${PORT:-$DEFAULT_PORT}"
}

listen_port() {
  local l="$1"
  printf '%s' "${l##*:}"
}

probe_host() {
  case "$1" in
    :*|0.0.0.0:*) echo 127.0.0.1 ;;
    \[*\]:*)      echo '[::1]' ;;
    *)            echo "${1%:*}" ;;
  esac
}

write_conf() {
  local listen="$1" token
  if [ -f "$CONF" ]; then
    # 升级路径：配置和令牌原样保留
    ok "沿用已有配置 $CONF"
    return 0
  fi
  token=$("$BIN" secret hex 24)
  umask 077
  cat > "$CONF" <<EOF
{
  "listen": "$listen",
  "data": "$DATA_DIR/nodes.json",
  "admin_token": "$token"
}
EOF
  chmod 600 "$CONF"
  [ "$SVC_USER" = subpanel ] && chown subpanel:subpanel "$CONF"
  ok "生成配置 $CONF"
}

write_systemd_unit() {
  local user="$1" caps="" port
  port=$(listen_port "$(resolved_listen)")
  if printf '%s' "$port" | grep -Eq '^[0-9]+$' && [ "$port" -lt 1024 ]; then
    caps=$'AmbientCapabilities=CAP_NET_BIND_SERVICE\nCapabilityBoundingSet=CAP_NET_BIND_SERVICE'
  else
    caps='CapabilityBoundingSet='
  fi
  cat > "/etc/systemd/system/$SERVICE.service" <<EOF
[Unit]
Description=subpanel subscription panel
Documentation=https://github.com/$REPO
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$user
Group=$user
ExecStart=$BIN serve -conf $CONF
Restart=on-failure
RestartSec=2
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=$DATA_DIR
$caps

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable "$SERVICE" >/dev/null 2>&1 || true
  ok "写入 systemd 服务"
}

write_openrc_unit() {
  local user="$1"
  cat > "/etc/init.d/$SERVICE" <<EOF
#!/sbin/openrc-run
name="subpanel"
description="subpanel subscription panel"
command="$BIN"
command_args="serve -conf $CONF"
command_user="$user:$user"
command_background=true
pidfile="/run/\${RC_SVCNAME}.pid"
output_log="/var/log/subpanel.log"
error_log="/var/log/subpanel.log"

depend() {
  need net
  after firewall
}
EOF
  chmod 0755 "/etc/init.d/$SERVICE"
  rc-update add "$SERVICE" default >/dev/null 2>&1 || true
  ok "写入 OpenRC 服务"
}

stop_service() {
  case "$INIT" in
    systemd) systemctl stop "$SERVICE" >/dev/null 2>&1 || true ;;
    openrc)  rc-service "$SERVICE" stop >/dev/null 2>&1 || true ;;
  esac
}

start_service() {
  case "$INIT" in
    systemd) systemctl restart "$SERVICE" ;;
    openrc)  rc-service "$SERVICE" restart >/dev/null 2>&1 || rc-service "$SERVICE" start ;;
  esac
}

health_check() {
  local listen host port url
  listen=$(resolved_listen)
  host=$(probe_host "$listen")
  port=$(listen_port "$listen")
  url="http://$host:$port/healthz"
  for _ in $(seq 1 40); do
    if curl -fsS --max-time 1 "$url" >/dev/null 2>&1; then
      ok "服务已响应 $url"
      return 0
    fi
    sleep 0.25
  done
  warn "服务在 10 秒内没有响应 $url"
  case "$INIT" in
    systemd) dim "  查看日志: journalctl -u $SERVICE -n 50 --no-pager" ;;
    openrc)  dim "  查看日志: tail -n 50 /var/log/subpanel.log" ;;
  esac
  return 1
}

server_ip() {
  local ip=""
  ip=$(hostname -I 2>/dev/null | awk '{print $1}' || true)
  if [ -z "$ip" ]; then
    ip=$(ip route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src"){print $(i+1); exit}}' || true)
  fi
  [ -n "$ip" ] || ip="<服务器IP>"
  printf '%s' "$ip"
}

print_done() {
  local listen port token host ip
  listen=$(resolved_listen)
  port=$(listen_port "$listen")
  token=$(SUBPANEL_CONF="$CONF" "$BIN" config-get admin_token 2>/dev/null || true)
  ip=$(server_ip)
  host="$ip"
  [ "$listen" = ":${port}" ] || [ "$listen" = "0.0.0.0:${port}" ] || host=$(probe_host "$listen")

  echo
  printf '%s安装完成%s\n' "$C_B" "$C_END"
  echo
  printf '  面板地址   http://%s:%s\n' "$host" "$port"
  printf '  管理令牌   %s\n' "${token:-（见 $CONF）}"
  printf '  配置文件   %s\n' "$CONF"
  printf '  数据文件   %s\n' "$DATA_DIR/nodes.json"
  echo
  dim "  令牌用于登录面板，也是 /api/sets 的 X-Admin-Token"
  if [ "$INIT" = systemd ]; then
    dim "  状态: systemctl status $SERVICE    日志: journalctl -u $SERVICE -f"
  elif [ "$INIT" = openrc ]; then
    dim "  状态: rc-service $SERVICE status   日志: tail -f /var/log/subpanel.log"
  else
    dim "  未安装服务，手动启动: $BIN serve -conf $CONF"
  fi
  echo
  dim "  公网访问请放行 $port 端口，并建议套 HTTPS 反代：面板本身走明文 HTTP"
}

do_install() {
  need_root
  local arch init upgraded=0 user
  arch=$(detect_arch) || die "不支持的架构: $(uname -m)"
  init=$(detect_init)
  INIT="$init"

  if [ -x "$BIN" ] && [ -f "$CONF" ]; then
    upgraded=1
    say "检测到已安装，按升级处理（配置与令牌保留）"
  else
    say "安装 subpanel（linux/$arch）"
  fi

  TMPD=$(mktemp -d)
  trap 'rm -rf "$TMPD"' EXIT

  download_binary "$TMPD" "$arch"

  if ensure_user; then
    SVC_USER=subpanel
  else
    SVC_USER=root
    warn "未能创建 subpanel 用户，将以 root 运行服务"
  fi
  user="$SVC_USER"

  mkdir -p "$CONF_DIR" "$DATA_DIR"
  # 配置目录 root 拥有、服务组可进入；服务只读配置，不能改
  chmod 750 "$CONF_DIR"
  chmod 750 "$DATA_DIR"
  if [ "$user" = subpanel ]; then
    chown root:subpanel "$CONF_DIR"
    chown subpanel:subpanel "$DATA_DIR"
  fi

  if [ "$upgraded" = 1 ]; then
    stop_service
  fi

  mkdir -p "$(dirname "$BIN")"
  install -m 0755 "$TMPD/subpanel-linux-$arch" "$BIN.new"
  mv -f "$BIN.new" "$BIN"
  ok "安装到 $BIN（$("$BIN" version)）"

  write_conf "$(resolved_listen)"

  case "$init" in
    systemd) write_systemd_unit "$user" ;;
    openrc)  write_openrc_unit "$user" ;;
    none)    warn "没有检测到 systemd/OpenRC，跳过服务安装" ;;
  esac

  if [ "$init" != none ]; then
    start_service || true
    health_check || true
  fi

  print_done
}

do_uninstall() {
  need_root
  INIT=$(detect_init)
  local had=0

  if [ -x "$BIN" ]; then had=1; fi
  if [ "$had" = 0 ] && [ ! -f "$CONF" ]; then
    say "没有检测到已安装的 subpanel"
    exit 0
  fi

  say "卸载 subpanel"
  case "$INIT" in
    systemd)
      systemctl stop "$SERVICE" >/dev/null 2>&1 || true
      systemctl disable "$SERVICE" >/dev/null 2>&1 || true
      rm -f "/etc/systemd/system/$SERVICE.service"
      systemctl daemon-reload
      ;;
    openrc)
      rc-service "$SERVICE" stop >/dev/null 2>&1 || true
      rc-update del "$SERVICE" default >/dev/null 2>&1 || true
      rm -f "/etc/init.d/$SERVICE"
      ;;
  esac
  ok "已停止并移除服务"

  if [ -x "$BIN" ]; then
    rm -f "$BIN"
    ok "已删除 $BIN"
  fi

  if [ "$PURGE" = 1 ]; then
    rm -rf "$CONF_DIR" "$DATA_DIR"
    ok "已删除配置与数据"
    if id subpanel >/dev/null 2>&1; then
      userdel subpanel 2>/dev/null || true
    fi
  else
    dim "  配置与数据保留在 $CONF_DIR 和 $DATA_DIR"
    dim "  彻底清理请加 --purge"
  fi
  echo
  say "卸载完成"
}

main() {
  parse_args "$@"
  case "$ACTION" in
    install)   do_install ;;
    uninstall) do_uninstall ;;
  esac
}

main "$@"
