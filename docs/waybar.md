# Waybar 配置示例

## 1. 添加模块到 Waybar config

在 `~/.config/waybar/config` 中添加：

```json
"custom/clashtui": {
  "exec": "clashtui --status",
  "on-click": "clashtui",
  "on-click-right": "clashtui --toggle",
  "interval": 5,
  "return-type": "json"
}
```

## 2. 添加样式

在 `~/.config/waybar/style.css` 中添加：

```css
#custom-clashtui {
  padding: 0 8px;
}

#custom-clashtui.running {
  color: #10b981;
}

#custom-clashtui.stopped {
  color: #6b7280;
}
```

## 3. 添加到模块列表

在 config 的 `"modules-left"` 或 `"modules-right"` 中加入 `"custom/clashtui"`：

```json
"modules-right": ["custom/clashtui", "clock", "tray"]
```

## 使用说明

| 操作 | 功能 |
|------|------|
| 左键点击 | 打开完整 TUI 界面 |
| 右键点击 | 快速开关代理 |

## 命令行选项

```bash
clashtui              # 打开 TUI 界面
clashtui --status     # 输出状态（供 Waybar 使用）
clashtui --daemon     # 后台运行（不显示界面）
clashtui --toggle     # 快速开关代理
clashtui --stop       # 停止代理
```