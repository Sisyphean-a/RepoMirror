---
doc_type: learning
track: knowledge
date: 2026-06-19
updated: 2026-06-19
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

# 反例补充

`internal/gitops/ignore.go` 里尝试过把 `parseIgnoredPathSet` 改成“先整体扫描一遍输出，确认是否存在 `\r` 和 `\\`，再按快路径逐行解析”。这个思路看起来能减少每行重复判断，但实际 benchmark 明显回退：

- `BenchmarkIgnoredPathSetFromRootSorted` 基线大致在 `42-46 us/op`
- 改后回退到大致 `55-62 us/op`

说明这类“先做一遍全量预扫描再走条件分支”的优化，在当前数据规模下额外遍历成本高于省下来的局部判断成本。

# 停止条件

如果一个热点满足下面两个条件，就该停止继续细抠并结束这轮优化：

1. benchmark 已经能稳定证明新尝试回退，而不是只是噪声波动。
2. 下一步候选改法已经不再是单文件、单热点、可快速自证的小改。

这时应当回到最近的收益提交，补文档，结束本轮，而不是为了“再挤一点”继续堆试验。
