---
doc_type: issue-fix
issue: 2026-06-22-push-button-stays-enabled-after-push
path: fast-track
fix_date: 2026-06-22
tags: [ui, git, push]
---

# 推送按钮已推送后仍可点击 修复记录

## 1. 问题描述

目标仓库在提交并成功推送后，界面上的“推送”按钮仍保持可点击，前端没有根据真实 Git 上游同步状态禁用该操作。

## 2. 根因

- `frontend/src/components/TargetStatusPanel.tsx` 的“推送”按钮只复用了通用禁用态，没有单独判断是否存在可推送提交。
- `internal/gitops/service.go` 解析目标仓库状态时只统计工作区改动，没有解析 `git status --porcelain=2 --branch` 返回的上游/领先落后信息，因此后端没有提供“是否可推送”的真实状态。

## 3. 修复方案

后端增加 `canPush` 状态，基于 `branch.upstream` 和 `branch.ab` 判断当前分支是否存在可执行普通 `git push` 的提交；前端改为使用该状态独立控制“推送”按钮禁用。

## 4. 改动文件清单

- `internal/model/types.go`
- `internal/gitops/service.go`
- `internal/gitops/service_test.go`
- `internal/app/service_test.go`
- `frontend/src/types.ts`
- `frontend/src/components/TargetStatusPanel.tsx`

## 5. 验证结果

- `go test ./...` 通过。
- `npm run build`（`frontend/`）通过。
- 新增回归测试覆盖：
  - 上游领先/已推送/无上游/分叉场景下的 `canPush` 判定。
  - 提交后可推送、推送后不可推送的服务层状态流转。

## 6. 遗留事项

无。
