#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="/opt/fast-note"
SERVICE_NAME="fast-note.service"
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
APP_SOURCE="$SCRIPT_DIR/fast-note-sync-service"
SEVENZIP_SOURCE="$SCRIPT_DIR/7zz"
APP_TARGET="$INSTALL_DIR/fast-note-sync-service"
SEVENZIP_TARGET="$INSTALL_DIR/7zz"
BACKUP_DIR="$INSTALL_DIR/test-backups/$(date -u +%Y%m%dT%H%M%SZ)"

if [ "$(id -u)" -ne 0 ]; then
  echo "请使用 sudo 运行：sudo ./install_encrypted_7z_test.sh" >&2
  exit 1
fi

if [ ! -d "$INSTALL_DIR" ] || [ ! -f "$APP_TARGET" ]; then
  echo "未找到现有的 fns 安装：$APP_TARGET" >&2
  exit 1
fi

if [ ! -f "$APP_SOURCE" ] || [ ! -f "$SEVENZIP_SOURCE" ]; then
  echo "测试包不完整，需要 fast-note-sync-service 和 7zz。" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$SCRIPT_DIR" && sha256sum -c SHA256SUMS)
fi

mkdir -p "$BACKUP_DIR"
cp -a "$APP_TARGET" "$BACKUP_DIR/fast-note-sync-service"
if [ -f "$SEVENZIP_TARGET" ]; then
  cp -a "$SEVENZIP_TARGET" "$BACKUP_DIR/7zz"
fi

stop_service() {
  if command -v fns >/dev/null 2>&1; then
    fns stop
  else
    systemctl stop "$SERVICE_NAME"
  fi
}

start_service() {
  if command -v fns >/dev/null 2>&1; then
    fns start
  else
    systemctl start "$SERVICE_NAME"
  fi
}

rollback() {
  echo "安装失败，正在恢复原程序……" >&2
  install -m 0755 "$BACKUP_DIR/fast-note-sync-service" "$APP_TARGET"
  if [ -f "$BACKUP_DIR/7zz" ]; then
    install -m 0755 "$BACKUP_DIR/7zz" "$SEVENZIP_TARGET"
  else
    rm -f "$SEVENZIP_TARGET"
  fi
  start_service || true
}

trap rollback ERR
stop_service
install -m 0755 "$APP_SOURCE" "$APP_TARGET"
install -m 0755 "$SEVENZIP_SOURCE" "$SEVENZIP_TARGET"
start_service
trap - ERR

echo "测试版本安装完成；config 和 storage 未修改。"
echo "原程序备份在：$BACKUP_DIR"
