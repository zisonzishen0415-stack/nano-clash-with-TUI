# ClashTUI

轻量级 Clash/mihomo 代理 TUI 管理工具，单文件编译，无外部依赖。

## 功能

- 📥 **订阅导入**：支持从剪贴板导入订阅链接，自动转换节点格式
- 📋 **节点管理**：显示节点列表、延迟测试、一键切换、自动选择最快
- ⚡ **快速操作**：一键启动/关闭内核，自动设置系统代理
- 📝 **实时日志**：所有操作和系统状态实时记录
- 🎨 **Vim风格**：j/k导航，h/l切换标签，全局快捷键
- 🔧 **多订阅管理**：支持多个订阅切换

## 支持协议

**节点链接格式：**
- Trojan (`trojan://`)
- VLESS (`vless://`)
- VMess (`vmess://`)
- Shadowsocks (`ss://`)
- ShadowsocksR (`ssr://`)
- Hysteria2 (`hysteria2://`, `hy2://`)
- Hysteria (`hysteria://`)
- SOCKS5 (`socks5://`, `socks://`)
- HTTP/HTTPS (`http://`, `https://`)
- WireGuard (`wireguard://`)
- TUIC (`tuic://`)
- SSH (`ssh://`)

**订阅格式：**
- HTTP/HTTPS URL（自动解析 base64 或 YAML）
- Clash YAML 配置文件

## 安装

```bash
# 编译
go build -o clashtui .

# 安装到 PATH
cp clashtui ~/.local/bin/
```

## 配置目录

```
~/.config/clashtui/
├── core/
│   └── clash          # mihomo 内核二进制
├── config.yaml        # 当前 Clash 配置
├── settings.json      # 用户设置（订阅、端口等）
├── Country.mmdb       # GeoIP 数据库
└── geosite.dat        # GeoSite 数据
```

## TUI 界面

三个标签页：

### 1. Nodes（节点）
显示所有代理节点，支持延迟测试和切换。

### 2. Config（配置）
- 订阅列表管理
- 设置选项（开机自启、自动测速、自动选择最快）
- 端口配置

### 3. Logs（日志）
实时显示所有操作和系统状态。

## 快捷键

### 全局快捷键

| 按键 | 功能 | 说明 |
|------|------|------|
| `1/2/3` 或 `h/l` | 切换标签页 | Nodes → Config → Logs |
| `r` | 重启内核 | 停止→下载订阅→启动→设置代理，每步都有日志反馈 |
| `x` | 停止内核 | 停止 mihomo + 清除系统代理，确保网络正常 |
| `c` | 导入订阅 | 从剪贴板读取订阅链接 |
| `s` | 添加订阅 | 手动输入订阅链接 |
| `q` 或 `ctrl+c` | 退出 | 安全退出，不影响正在运行的内核 |

### Nodes 标签页

| 按键 | 功能 |
|------|------|
| `j/k` 或 `↑/↓` | 选择节点 |
| `enter` | 切换到选中节点 |
| `t` | 测试当前节点延迟 |
| `T` | 测试所有节点延迟 |
| `x` | 停止内核（同全局） |

### Config 标签页

| 按键 | 功能 |
|------|------|
| `j/k` 或 `↑/↓` | 选择项目 |
| `enter` | 执行/切换选项 |
| `d` | 删除当前订阅 |
| `D` | 删除所有订阅 |

### Logs 标签页

只读，自动显示最新日志。

## 操作反馈

界面底部显示当前操作状态：
- `⏳ 操作名称` - 正在进行中
- `✓ 成功信息` - 操作成功
- `⚠ 错误信息` - 操作失败

所有操作同时写入 Logs 标签页。

## 命令行

| 命令 | 功能 |
|------|------|
| `clashtui` | 打开 TUI 界面 |
| `clashtui --status` | 输出状态 JSON（供 Waybar 使用） |
| `clashtui --toggle` | 快速开关代理 |
| `clashtui --stop` | 停止代理并清除设置 |
| `clashtui --daemon` | 后台模式运行 |

## Waybar 集成

```json
"custom/clashtui": {
  "exec": "clashtui --status",
  "on-click": "clashtui",
  "on-click-right": "clashtui --toggle",
  "interval": 5
}
```

状态 JSON 格式：
```json
{"text":"🟢","tooltip":"Proxy: 新加坡01","class":"running"}
```

详见 [docs/waybar.md](docs/waybar.md)

## 使用流程

### 导入订阅

1. 运行 `clashtui` 打开界面
2. 按 `2` 进入 Config 标签页
3. 选择 `[+] Add subscription/nodes`，按 `enter`
4. 选择导入方式：
   - `[1]` 订阅 - 手动输入 URL
   - `[2]` 订阅 - 从剪贴板导入 URL
   - `[3]` 节点 - 手动输入链接（支持多行）
   - `[4]` 节点 - 从剪贴板导入链接
5. 输入订阅/节点名称后保存
6. 按 `enter` 激活订阅，或按 `r` 全局启动

### 快捷导入

- 按 `c`：直接从剪贴板导入（自动识别订阅 URL 或节点链接）

### 使用代理

1. 按 `1` 返回 Nodes 标签页
2. 使用 `j/k` 选择节点，`enter` 切换
3. 按 `T` 测试所有节点延迟，自动选择最快（如果启用）

## 安全退出

- 按 `x` 后再按 `q`：停止内核并退出，系统网络正常
- 直接按 `q`：退出 TUI，内核继续运行（代理保持生效）

**重要**：如果需要完全停止代理，务必先按 `x` 再退出。

## 系统要求

- Linux (支持 Wayland/X11)
- Go 1.21+
- TUN 模式需要: `sudo setcap cap_net_admin+ep ~/.config/clashtui/core/clash`

## Clipboard 支持

- Wayland: 安装 `wl-clipboard`
- X11: 安装 `xclip` 或 `xsel`

## 开机自启

在 Config 标签页开启 "Auto start on boot" 选项，会创建 systemd user service：
- 服务名: `clashtui.service`
- 模式: `--daemon` 后台运行

禁用后自动删除服务文件。

## License

MIT