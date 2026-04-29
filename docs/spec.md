# ClashTUI - Clash 代理管理 TUI 工具

## 项目概述

轻量级 Clash 代理管理工具，提供 TUI 界面控制代理内核。

## 核心功能

### 1. 订阅导入
- 手动输入订阅链接
- 从剪贴板读取订阅链接（使用 xclip/xsel）
- 下载并保存配置到 `~/.config/clashtui/config.yaml`

### 2. 节点管理
- 列出所有代理节点
- 显示节点延迟（实时测速）
- 一键批量测速所有节点
- 选择节点切换

### 3. Clash 控制
- 启动/停止 Clash 内核
- 内置 mihomo 内核（首次启动自动下载到 `~/.config/clashtui/core/`)
- 内核 API 地址: `127.0.0.1:9090`
- 混合端口: `7890`

### 4. 单实例机制
- PID 文件: `/tmp/clashtui.pid`
- 启动时检查已有进程
- 若已运行则唤醒（发送 SIGUSR1），不启动新实例

## 技术栈

- **Go 1.21+** - 主程序
- **bubbletea** - TUI 框架
- **bubbles** - UI 组件（list, spinner, progress）
- **xclip/xsel** - 剪贴板读取（系统依赖）

## 数据目录结构

```
~/.config/clashtui/
├── core/
│   └── clash          # mihomo 内核二进制
├── config.yaml        # 当前 Clash 配置
├── Country.mmdb       # GeoIP 数据库
├── geosite.dat        # GeoSite 数据
└── subscription.txt   # 订阅链接缓存
```

## TUI 界面布局

标签页式设计，顶部切换：

```
┌──────────────────────────────────────────────────────────────┐
│ [节点] [配置] [日志]                                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  🇸🇬 新加坡01 | 专线 | 延迟: 200ms                             │
│  🇸🇬 新加坡02 | 专线 | 延迟: 150ms ✓                           │
│  🇯🇵 日本01   | 专线 | 延迟: 500ms                             │
│  ...                                                         │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│ 状态: 运行中 | 当前: 新加坡02 | 按 q 退出                       │
└──────────────────────────────────────────────────────────────┘
```

## 快捷键

| 键 | 功能 |
|----|------|
| `j/k` 或 `↑/↓` | 上下选择节点 |
| `Enter` | 切换到选中节点 |
| `r` | 刷新/测速当前节点 |
| `R` | 一键测速所有节点 |
| `s` | 导入订阅 |
| `c` | 从剪贴板导入 |
| `p` | 启动/停止 Clash |
| `l` | 切换到日志标签 |
| `q` | 退出 |

## API 设计

工具通过 Clash RESTful API 控制内核：

```bash
# 获取节点列表
GET http://127.0.0.1:9090/proxies

# 切换节点
PUT http://127.0.0.1:9090/proxies/GLOBAL
Body: {"name": "节点名称"}

# 测速节点
GET http://127.0.0.1:9090/proxies/{节点名称}/delay?timeout=5000&url=http://www.gstatic.com/generate_204
```

## 内核下载

首次启动自动下载 mihomo 内核：
- 源: `https://gh-proxy.com/https://github.com/MetaCubeX/mihomo/releases/latest`
- 文件: `mihomo-linux-amd64.gz`
- 解压到: `~/.config/clashtui/core/clash`

## 成功标准

1. 单二进制文件，无需额外依赖（除 xclip/xsel）
2. 启动后能正常显示节点列表
3. 能切换节点并生效
4. 测速功能正常工作
5. 单实例机制正常工作