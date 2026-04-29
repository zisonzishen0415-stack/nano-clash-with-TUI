# Nano Clash TUI

轻量级 Clash 代理 TUI 管理工具，单文件编译，无外部依赖。

## 功能

- 📥 **订阅导入**：支持从剪贴板导入订阅链接，自动转换节点格式
- 📋 **节点管理**：显示节点列表、延迟测试、一键切换
- ⚡ **快速操作**：一键启动/关闭内核，自动设置系统代理
- 🎨 **Vim风格**：j/k导航，h/l切换标签，全局快捷键

## 支持协议

- Trojan
- VLESS
- Hysteria2

## 快捷键

| 按键 | 功能 |
|------|------|
| `1/2/3` 或 `h/l` | 切换标签页 |
| `j/k` | 选择节点 |
| `enter` | 切换节点 |
| `t` | 测试当前节点延迟 |
| `T` | 测试所有节点延迟 |
| `r` | 刷新订阅并启动内核 |
| `c` | 从剪贴板导入订阅 |
| `x` | 关闭内核并清除代理 |
| `q` | 退出 |

## 安装

```bash
# 编译
go build -o clashtui .

# 运行 TUI
./clashtui
```

## Waybar 集成

支持在 Waybar 状态栏显示代理状态：

```json
"custom/clashtui": {
  "exec": "clashtui --status",
  "on-click": "clashtui",
  "on-click-right": "clashtui --toggle",
  "interval": 5
}
```

详见 [docs/waybar.md](docs/waybar.md)

## 命令行

| 命令 | 功能 |
|------|------|
| `clashtui` | 打开 TUI 界面 |
| `clashtui --status` | 输出状态 JSON |
| `clashtui --toggle` | 快速开关代理 |
| `clashtui --stop` | 停止代理 |

## 使用

1. 按 `2` 进入 Config 标签页
2. 按 `c` 从剪贴板导入订阅链接
3. 按 `r` 启动内核
4. 按 `1` 返回 Nodes 标签页，选择节点

## 系统要求

- Linux (支持 Wayland/X11)
- Go 1.21+
- TUN模式需要: `sudo setcap cap_net_admin+ep ~/.config/clashtui/core/clash`

## Clipboard支持

- Wayland: 安装 `wl-clipboard`
- X11: 安装 `xclip` 或 `xsel`

## License

MIT
