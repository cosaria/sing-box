#!/usr/bin/env bash
set -euo pipefail

# sing-box panel 安装脚本
# 用法: bash <(curl -sL https://raw.githubusercontent.com/cosaria/sing-box/main/install.sh)

REPO="cosaria/sing-box"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/usr/local/etc/sing-box"
BIN_NAME="sing-box"
LOCAL_INSTALL=false
VERSION=""
PROXY=""

usage() {
    echo "用法: bash install.sh [选项]"
    echo ""
    echo "选项:"
    echo "  -l          本地安装（从当前目录编译）"
    echo "  -v VERSION  指定版本（如 v0.1.0）"
    echo "  -p PROXY    使用代理下载（如 http://127.0.0.1:2333）"
    echo "  -h          显示帮助"
}

detect_platform() {
    local os arch
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    if [[ "$os" != "linux" ]]; then
        echo "错误: 仅支持 Linux 系统" >&2
        exit 1
    fi

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)
            echo "错误: 不支持的架构 $arch" >&2
            exit 1
            ;;
    esac

    echo "$os/$arch"
}

get_latest_version() {
    local url="https://api.github.com/repos/$REPO/releases/latest"
    if [[ -n "$PROXY" ]]; then
        curl -sL --proxy "$PROXY" "$url" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/'
    else
        curl -sL "$url" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/'
    fi
}

download_binary() {
    local version="$1" arch="$2"
    local filename="${BIN_NAME}-linux-${arch}"
    local url="https://github.com/$REPO/releases/download/$version/$filename"

    echo "下载 $url ..."

    local curl_opts=(-sL --fail -o "$INSTALL_DIR/$BIN_NAME")
    if [[ -n "$PROXY" ]]; then
        curl_opts+=(--proxy "$PROXY")
    fi

    if ! curl "${curl_opts[@]}" "$url"; then
        echo "错误: 下载失败" >&2
        exit 1
    fi

    chmod +x "$INSTALL_DIR/$BIN_NAME"
}

local_install() {
    echo "本地编译安装..."
    if ! command -v go &>/dev/null; then
        echo "错误: 未找到 go 命令" >&2
        exit 1
    fi
    CGO_ENABLED=0 go build -o "$INSTALL_DIR/$BIN_NAME" ./cmd/sing-box/
    chmod +x "$INSTALL_DIR/$BIN_NAME"
}

main() {
    while getopts "lv:p:h" opt; do
        case "$opt" in
            l) LOCAL_INSTALL=true ;;
            v) VERSION="$OPTARG" ;;
            p) PROXY="$OPTARG" ;;
            h) usage; exit 0 ;;
            *) usage; exit 1 ;;
        esac
    done

    if [[ $EUID -ne 0 ]]; then
        echo "错误: 请使用 root 权限运行" >&2
        exit 1
    fi

    local platform
    platform=$(detect_platform)
    local arch="${platform#*/}"
    echo "平台: $platform"

    if [[ "$LOCAL_INSTALL" == true ]]; then
        local_install
    else
        if [[ -z "$VERSION" ]]; then
            VERSION=$(get_latest_version)
            if [[ -z "$VERSION" ]]; then
                echo "错误: 无法获取最新版本" >&2
                exit 1
            fi
        fi
        echo "版本: $VERSION"
        download_binary "$VERSION" "$arch"
    fi

    # 创建数据目录
    mkdir -p "$DATA_DIR"

    # 安装系统服务
    echo "安装系统服务..."
    "$INSTALL_DIR/$BIN_NAME" service install

    # 启动服务
    echo "启动服务..."
    if command -v systemctl &>/dev/null; then
        systemctl start sing-box
    elif command -v rc-service &>/dev/null; then
        rc-service sing-box start
    fi

    sleep 2

    # 显示信息
    echo ""
    echo "========================================="
    echo " sing-box panel 安装完成"
    echo "========================================="
    echo ""
    echo "管理命令: sing-box"
    echo "守护进程: sing-box serve"
    echo "查看状态: sing-box version"
    echo "更新版本: sing-box update"
    echo ""
}

main "$@"
