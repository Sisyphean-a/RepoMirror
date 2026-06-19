---
doc_type: feature-ff-note
feature: add-ai-commit-message
date: 2026-06-19
requirement:
tags: [git, ai, commit, deepseek]
---

## 做了什么
给目标仓库提交区增加了一个最小版 AI 生成提交信息能力，直接使用 DeepSeek 根据当前目标仓库变更生成一条中文提交信息。
支持在应用内保存 DeepSeek API Key，并在后端持久化后复用。

## 改了哪些
- `internal/aicommit/*` — 新增 DeepSeek 调用与响应解析
- `internal/gitops/commit_summary.go` — 新增目标仓库工作区摘要提取
- `internal/app/ai_commit.go` — 新增生成提交信息与保存 API Key 的应用层动作
- `internal/model/types.go` `internal/app/state.go` — 配置持久化 Key，但返回前端时脱敏，只暴露是否已配置
- `frontend/src/useRepoMirror.ts` `frontend/src/components/TargetStatusPanel.tsx` — 增加 Key 保存与 AI 生成交互

## 怎么验证的
执行了 `go test ./...`、`go build ./...` 和 `frontend` 下的 `npm run build`，均通过。
新增了 DeepSeek 客户端、工作区摘要、应用层生成/持久化的自动化测试。
