---
doc_type: issue-fix
issue: 2026-06-18-windows-git-console-and-config-persist
path: fast-track
fix_date: 2026-06-19
tags: [windows, persistence, ui]
---

# Windows Git 弹窗与配置不落盘修复记录

## 1. 问题描述

- 选择仓库、交换仓库和切换同步方向后，配置不会立即落盘，重启应用后需要重新选择仓库。
- 顶部仓库信息区域分成两行，纵向空间占用偏大。
- Windows 下每次选择仓库或刷新状态都会为 `git` 子进程闪出控制台窗口。

## 2. 根因

- `internal/app/actions.go` 只更新内存里的 `AppConfig`，磁盘写入仍依赖手动“保存配置”或窗口关闭时机。
- `frontend/src/components/AppHeader.tsx` 把 A/B 仓库栏拆成上下两行渲染。
- `internal/gitops/service.go` 直接用 `exec.Command("git", ...)` 启动子进程，Windows 默认会附带控制台窗口。

## 3. 修复方案

- 配置变更路径改为“保存成功后再提交到内存状态”；保存失败直接返回错误，关窗保存失败则写 Wails 错误日志，不再静默吞掉。
- 头部仓库栏改为单行布局，并同步调整分隔线与窄屏换行样式。
- 为 Windows 的 `git` 子进程补 `HideWindow` / `CREATE_NO_WINDOW`，非 Windows 保持空实现。

## 4. 改动文件清单

- `frontend/src/components/AppHeader.tsx`
- `frontend/src/styles/header.css`
- `frontend/src/styles/shell.css`
- `internal/app/actions.go`
- `internal/app/service.go`
- `internal/app/service_test.go`
- `internal/gitops/service.go`
- `internal/gitops/hide_window_windows.go`
- `internal/gitops/hide_window_other.go`

## 5. 验证结果

- `go test ./...`
- `npm run build`
- 新增自动持久化回归测试，覆盖仓库选择 / 交换 / 方向切换自动落盘，以及持久化失败时内存配置不被污染。

## 6. 遗留事项

- 未在当前会话里做交互式桌面验收；Windows 控制台闪窗修复基于子进程创建参数与本地编译验证。
