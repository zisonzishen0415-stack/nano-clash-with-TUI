#!/bin/bash
# ClashTUI 安装脚本
# 用法: curl -fsSL https://raw.githubusercontent.com/.../install.sh | bash
# 或者: ./install.sh

set -e

REPO_URL="https://github.com/zisonz/clashtui"
INSTALL_DIR="$HOME/.local/bin"
CONFIG_DIR="$HOME/.config/clashtui"

echo "=== ClashTUI 安装 ==="

# 检查依赖
echo "[1/4] 检查依赖..."
MISSING=""
for cmd in go curl; do
    if ! command -v $cmd &>/dev/null; then
        MISSING="$MISSING $cmd"
    fi
done

if [ -n "$MISSING" ]; then
    echo "错误: 缺少依赖 $MISSING"
    echo "请安装: sudo apt install $MISSING (或对应包管理器)"
    exit 1
fi

# 创建目录
echo "[2/4] 创建目录..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"

# 下载或克隆
echo "[3/4] 获取源码..."
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

if [ -d "$HOME/development/clashtui" ]; then
    echo "使用本地源码: $HOME/development/clashtui"
    cd "$HOME/development/clashtui"
else
    echo "从 GitHub 克隆..."
    git clone --depth 1 "$REPO_URL" .
fi

# 编译安装
echo "[4/4] 编译安装..."
go build -o clashtui .
cp clashtui "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/clashtui"

# 清理
cd /
rm -rf "$TEMP_DIR"

# 检查 PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "警告: $INSTALL_DIR 不在 PATH 中"
    echo "请添加以下内容到 ~/.bashrc 或 ~/.zshrc:"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

# 完成
echo ""
echo "=== 安装完成 ==="
echo ""
echo "使用方法:"
echo "  clashtui          启动 TUI"
echo "  clashtui --restore-network   恢复网络（当网络中断时使用）"
echo "  clashtui --stop   停止内核并清除代理"
echo "  clashtui --toggle 切换代理开关"
echo ""
echo "首次使用:"
echo "  1. 运行 clashtui"
echo "  2. 按 2 切换到 Config 标签"
echo "  3. 按 c 从剪贴板导入订阅链接"
echo ""