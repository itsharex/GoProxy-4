# Phase 3 功能完成情况清单

> 对照来源：`docs/phase-03-auth-stats-tray.md`  
> 检查日期：2026-04-28

## 总览

Phase 3 目标包括认证管理、实时流量统计、可视化图表、系统托盘和主题设置。当前项目已经完成了部分流量统计、连接明细、仪表盘实时图表和主题切换；认证、系统托盘、正式统计事件推送和 ECharts 统计页仍未完成。

## 任务清单

| 编号 | 任务 | 状态 | 当前情况 |
|------|------|------|----------|
| P3-01 | 实现 Auth Module | 未完成 | `internal/proxy/auth.go` 仍是 `NoopAuthenticator` 占位；配置结构中还没有 `auth` 字段；未实现 bcrypt 用户管理。 |
| P3-02 | 接入 SOCKS5 认证 | 未完成 | SOCKS5 当前只支持 no-auth；未实现 RFC 1929 username/password 子协商。 |
| P3-03 | 接入 HTTP CONNECT 认证 | 未完成 | HTTP CONNECT 未校验 `Proxy-Authorization`；目前仅删除转发请求中的 `Proxy-Authorization` 头，未做认证逻辑。 |
| P3-04 | 实现认证管理绑定方法 | 未完成 | `app.go` 尚未提供 `AddUser`、`RemoveUser`、密码重置等 Wails 绑定方法。 |
| P3-05 | 实现认证管理页 | 未完成 | 左侧菜单中“认证管理”仍为 disabled；没有认证开关、用户表格、新增、删除或重置密码 UI。 |
| P3-06 | 完善 StatsCollector | 部分完成 | `internal/stats` 已用 atomic 统计活跃连接、总连接、上下行累计字节；尚未在后端 `Stats` 中提供 `uploadRate`、`downloadRate`、`authFailures` 字段。 |
| P3-07 | 接入 Relay 字节统计 | 已完成 | Relay 已接入上下行字节统计，并支持连接级活跃连接快照；近期已做批量刷新以降低吞吐影响。 |
| P3-08 | 实现统计事件推送 | 未完成 | 当前前端通过 `GetStats()` 轮询刷新；后端尚未每秒 `EventsEmit("proxy:stats", ...)`。 |
| P3-09 | 实现图表页面 | 部分完成 | 仪表盘已有近 60 秒实时流量曲线、坐标轴、hover 提示和统计卡片；但未使用 ECharts，独立“流量统计”页仍 disabled，协议分布/认证失败等统计未实现。 |
| P3-10 | 实现系统托盘 | 未完成 | `internal/platform` 目前只有平台路径；未实现托盘图标、右键菜单、双击恢复、最小化到托盘。 |
| P3-11 | 实现主题设置 | 部分完成 | 前端已支持 light/dark/auto 主题读取和切换，并保存到 YAML；但独立“应用设置”页仍 disabled，开机自启、最小化到托盘、语言设置等未完成。 |

## 已完成内容

- 基础统计：活跃连接、总连接、上行累计、下行累计。
- Relay 上下行字节统计。
- 活跃连接详情快照：协议、客户端、目标、上下行流量、建立时间。
- 独立“活跃连接”菜单页。
- 仪表盘实时流量曲线，包含上传/下载图例、坐标轴刻度和鼠标 hover 数值提示。
- 仪表盘客户端聚合列表：按客户端 IP 聚合连接数、实时上下行和总上下行。
- 前端主题切换：亮色、暗色、跟随系统。

## 未完成重点

- 认证系统整体未开始落地：配置、bcrypt、SOCKS5 RFC 1929、HTTP Basic Proxy-Authorization、认证管理 UI 都未完成。
- 后端统计事件推送未完成：缺少 `proxy:stats` 每秒事件。
- 后端统计快照字段不完整：缺少上传速率、下载速率、认证失败次数。
- 独立流量统计页未完成：菜单仍禁用，未实现 ECharts 页面和协议分布。
- 系统托盘未完成：托盘图标、菜单、状态同步、窗口显示/隐藏都未实现。
- 应用设置页未完成：主题以外的设置项尚无完整 UI。

## 建议下一步顺序

1. 先完成 Auth 配置结构和 `AuthManager`，补 bcrypt hash、用户唯一性校验和单元测试。
2. 接入 SOCKS5 RFC 1929 与 HTTP CONNECT Basic 认证。
3. 增加认证管理 Wails 绑定和页面。
4. 扩展 `Stats` 字段，后端每秒推送 `proxy:stats`。
5. 实现独立流量统计页。
6. 实现 `internal/platform` 托盘模块，并处理 Windows/macOS 差异。
