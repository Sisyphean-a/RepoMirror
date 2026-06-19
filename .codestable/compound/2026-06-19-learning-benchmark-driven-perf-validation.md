---
doc_type: learning
track: knowledge
date: 2026-06-19
slug: benchmark-driven-perf-validation
component: performance-workflow
tags:
  - benchmark
  - pprof
  - gitops
  - app
  - optimization
---

# 背景

这轮性能优化同时覆盖了 `internal/syncer`、`internal/gitops`、`internal/app` 和前端渲染路径。有效优化和无效优化都很多，单靠直觉很容易把回退改动混进主线。

# 指导原则

每一刀优化都要按“单点改动 -> 单点 benchmark -> 必要时 profile -> 立即提交或立即回退”执行，不攒多笔改动一起验证。

# 为什么重要

这轮里有两类很典型的信号：

1. `internal/gitops/service.go` 的路径规整优化虽然改动很小，但 benchmark 直接把 `BenchmarkBuildTargetStatus` 从 `16 B/op, 2 allocs/op` 压到 `8 B/op, 1 alloc/op`，`BenchmarkReadTargetStatusFromRoot` 也降到 `3 allocs/op`，属于应当立刻提交的净收益。
2. `internal/app/state.go` 和后续一次 `buildTargetStatus` 顺序扫描重构，从代码直觉看像是在“减少重复扫描/减少指针共享”，但 benchmark 明确回退，说明 goroutine 调度、`WaitGroup`、运行时成本比这类局部重构更敏感；这种改动必须当场回掉，不能凭感觉保留。

# 何时适用

适用于这类优化任务：

- 目标是压 `ns/op`、`B/op`、`allocs/op`
- 热点已能用 benchmark 或 pprof 稳定复现
- 改动可以收敛到单文件或单热点
- 存在“看起来更优、实际可能更慢”的风险

# 示例

本次工作里可复用的执行顺序：

1. 先做单文件小改，例如 `internal/gitops/service.go` 的路径规整。
2. 立即跑定向 benchmark，而不是先扩到其他模块。
3. benchmark 不回退时，再补 profile 看热点是否仍一致。
4. 结果成立就立刻提交单独 commit。
5. 如果 benchmark 回退，即使分配不变、代码更“整洁”，也直接回退，不混入后续提交。
