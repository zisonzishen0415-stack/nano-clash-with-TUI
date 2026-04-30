# ClashTUI

轻量级 Clash/mihomo 代理 TUI 管理工具，单文件编译，无外部依赖。

## 功能

- 📥 **订阅导入**：支持从剪贴板导入订阅链接，自动转换节点格式
- 📋 **节点管理**：显示节点列表、延迟测试、一键切换、自动选择最快
- ⚡ **快速操作**：一键启动/关闭内核，自动设置系统代理（GNOME/KDE）
- 📝 **实时日志**：所有操作和系统状态实时记录
- 🎨 **Vim风格**：j/k导航，h/l切换标签，全局快捷键
- 🔧 **多订阅管理**：支持多个订阅切换
- 🔄 **终端自动同步**：首次运行自动配置 ~/.bashrc，新终端自动加载代理状态

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
# 编译并安装
go build -o clashtui .
cp clashtui ~/.local/bin/
```

首次运行时会自动配置 `~/.bashrc` 和 `~/.zshrc`，新开的终端会自动加载代理状态。

## 配置目录

```
~/.config/clashtui/
├── core/
│   └── clash          # mihomo 内核二进制
├── config.yaml        # 当前 Clash 配置
├── settings.json      # 用户设置（订阅、端口等）
├── proxy.sh           # 终端代理脚本（自动生成）
├── Country.mmdb       # GeoIP 数据库
└── geosite.dat        # GeoSite 数据
```

## TUI 界面

三个标签页：

### 1. Nodes（节点）
显示所有代理节点，支持延迟测试和切换。

### 2. Config（配置）
- 订阅列表管理
- 设置选项：开机自启、自动测速、自动选择最快、**System proxy**
- 端口配置

### 3. Logs（日志）
实时显示所有操作和系统状态。

## 快捷键

### 全局快捷键

| 按键 | 功能 | 说明 |
|------|------|------|
| `1/2/3` 或 `h/l` | 切换标签页 | Nodes → Config → Logs |
| `r` | 重启内核 | 重新下载订阅并启动 |
| `x` | 停止内核 | 停止代理，清除系统代理，留在 TUI |
| `c` | 导入订阅 | 从剪贴板读取订阅链接 |
| `s` | 添加订阅 | 手动输入订阅链接 |
| `q` 或 `ctrl+c` | 退出 TUI | 退出程序，内核继续运行 |

### Nodes 标签页

| 按键 | 功能 |
|------|------|
| `j/k` 或 `↑/↓` | 选择节点 |
| `enter` | 切换到选中节点 |
| `t` | 测试当前节点延迟 |
| `T` | 测试所有节点延迟 |

### Config 标签页

| 按键 | 功能 |
|------|------|
| `j/k` 或 `↑/↓` | 选择项目 |
| `enter` | 执行/切换选项 |
| `d` | 删除当前订阅 |
| `D` | 删除所有订阅 |

### Logs 标签页

只读，自动显示最新日志。

## 命令行

| 命令 | 功能 |
|------|------|
| `clashtui` | 打开 TUI 界面 |
| `clashtui --status` | 输出状态 JSON（供 Waybar 使用） |
| `clashtui --toggle` | 快速开关代理 |
| `clashtui --stop` | 停止代理并清除系统代理 |
| `clashtui --daemon` | 后台模式运行 |
| `clashtui --env` | 打印代理环境变量 |

## 代理生效范围

### GUI 应用（浏览器等）

在 Config 标签页开启 **System proxy** 选项：

**Firefox/大多数应用：**
- 通过 GNOME/KDE gsettings 自动生效
- 从 wofi/桌面图标启动的应用自动使用代理

**Chrome/Chromium（Wayland）：**
- Chrome 在 Wayland 下忽略 gsettings 代理设置
- 开启 System proxy 时自动创建 `chrome-proxy.desktop` 启动器
- 在 wofi 中选择 "Chrome (Proxy)" 即可使用代理访问外网
- 关闭 System proxy 时自动删除该启动器

**清理：**
- 按 `x` 或运行 `--stop` 时自动清除系统代理和 DNS 缓存

### 终端应用

首次运行 clashtui 后，自动在 `~/.bashrc` 和 `~/.zshrc` 添加：

```bash
[ -f ~/.config/clashtui/proxy.sh ] && source ~/.config/clashtui/proxy.sh 2>/dev/null
```

**效果：**
- 新开的终端（foot、bash、zsh）自动加载当前代理状态
- 启用代理时：自动设置 `HTTP_PROXY` 等环境变量
- 禁用代理时：自动清除环境变量

**当前终端需手动同步：**
```bash
source ~/.config/clashtui/proxy.sh
# 输出：Proxy enabled 或 Proxy disabled
```

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
{"text":"🟢","tooltip":"Proxy: DIRECT","class":"running"}
{"text":"🔴","tooltip":"Proxy: stopped","class":"stopped"}
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
6. 按 `enter` 激活订阅

### 快捷导入

按 `c` 直接从剪贴板导入（自动识别订阅 URL 或节点链接）

### 使用代理

1. 在 Config 开启 **System proxy**
2. 按 `1` 返回 Nodes 标签页
3. `j/k` 选择节点，`enter` 切换
4. 按 `T` 测试所有节点延迟

## 系统要求

- Linux (支持 Wayland/X11)
- Go 1.21+
- TUN 模式需要: `sudo setcap cap_net_admin+ep ~/.config/clashtui/core/clash`

## Clipboard 支持

- Wayland: 安装 `wl-clipboard`
- X11: 安装 `xclip` 或 `xsel`

## 开机自启

在 Config 标签页开启 "Auto start on boot"，创建 systemd user service：
- 服务名: `clashtui.service`
- 模式: `--daemon` 后台运行

禁用后自动删除服务文件。

## License

MIT