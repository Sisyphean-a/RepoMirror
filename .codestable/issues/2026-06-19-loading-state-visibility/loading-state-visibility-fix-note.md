---
doc_type: issue-fix
issue: 2026-06-19-loading-state-visibility
path: fast-track
fix_date: 2026-06-19
tags: [frontend, loading, feedback]
---

# Loading 状态不可见修复记录

## 1. 问题描述

- 除首次启动外，刷新、选仓库、交换仓库、切换方向、保存配置、同步、提交、推送都没有可见 loading 提示。
- 当前实现只有按钮 `disabled`，用户无法判断操作是否正在执行。

## 2. 根因

- `frontend/src/useRepoMirror.ts` 只维护布尔型 `busy`，没有记录“正在做什么”。
- `frontend/src/App.tsx` 仅在 `state === null` 时渲染初始 `LoadingScreen`，正常工作区没有任何进行中反馈。
- `frontend/src/components/AppStatusBar.tsx` 只显示错误/成功/就绪，不展示进行中状态。

## 3. 修复方案

- 为前端状态机增加 `busyMessage`，所有异步动作都写明进行中的操作文案。
- 在工作区增加统一 loading 蒙层，在初始加载页和底部状态栏复用同一套进行中反馈。
- 保留原有按钮禁用逻辑，但不再只靠禁用态表达“正在执行”。

## 4. 改动文件清单

- `frontend/src/useRepoMirror.ts`
- `frontend/src/App.tsx`
- `frontend/src/components/AppStatusBar.tsx`
- `frontend/src/styles/shell.css`
- `frontend/src/styles/status-bar.css`

## 5. 验证结果

- `npm run build`

## 6. 遗留事项

- 本次未做桌面端交互验收，仅完成静态构建验证。
