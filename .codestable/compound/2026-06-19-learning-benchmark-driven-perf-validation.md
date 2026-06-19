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

# 高抖动基准的对照方法

有些 benchmark 单次波动很大，不能只看“改后跑了 5 次里最好的一组”。这时应当在同一会话里做成对对照：

1. 先保留候选实现，跑固定次数并落盘。
2. 回到基线实现，用完全相同的命令再跑一遍并落盘。
3. 直接比较候选和基线的均值、中位数，而不是只比单次最优值。

这次 `internal/gitops/paths.go` 的 `.git` 候选早返回就是按这个方法确认的：

- candidate: `average 157141.6 ns/op`, `median 171005.5 ns/op`
- baseline: `average 182023.4 ns/op`, `median 195249.5 ns/op`

这种成对对照比“改前隔几轮跑一次、改后再凭印象比较”可靠得多，特别适合高抖动的 IO / 并发相关基准。

# 字符热点的进一步收紧

如果 profile 已经明确显示热点落在某个“逐字符扫整串”的辅助函数里，可以继续用更便宜的定位原语缩小扫描范围，而不是重写整条主流程。

这次在 `internal/gitops/paths.go` 里，`hasProtectedPathCandidate` 原本会逐字符判断整条路径是否存在段首 `.`。profile 说明它本身已经成了次级热点后，可以继续把它改成：

1. 先用 `strings.IndexByte(..., '.')` 只跳到真正的 `.` 位置。
2. 只在这些命中的位置上检查“是否位于段首”和“是否可能是 `.git` 前缀”。

同 session 的候选/基线 10 次对照结果：

- candidate: `average 151570.8 ns/op`, `median 159163 ns/op`
- baseline: `average 161255.7 ns/op`, `median 173760 ns/op`

要点不是“看到字符串就上 `IndexByte`”，而是先确认热点确实来自大量无效字符扫描，再把线性扫整串改成“只扫目标字符命中点”。

# 预分配写入不一定优于 append

即使 profile 里 `buildSingleGroupInputWithoutDedup` 仍然占了可见 CPU，也不能直接把它改成“先按总长度扩容，再用 `copy` + 写换行符做定长写入”。

这次在 `internal/gitops/service.go` 的单组有序路径上试过把：

1. `analyzeSingleGroupPaths` 识别出的“非空、严格升序”输入走专用快路径。
2. 快路径改成 `buffer = buffer[:totalBytes]` 后按偏移量 `copy` 整段路径，再手写 `'\n'`。

同 session 的 5 次候选/基线对照结果没有形成稳定净收益：

- `BenchmarkIgnoredPathsFromRoot`
- candidate: `average 71325.8 ns/op`, `median 78052 ns/op`
- baseline: `average 74378.8 ns/op`, `median 82466 ns/op`
- `BenchmarkIgnoredPathSetFromRootSorted`
- candidate: `average 44075.8 ns/op`, `median 43478 ns/op`
- baseline: `average 44552.6 ns/op`, `median 42474 ns/op`

第一条 benchmark 看起来略快，但第二条的中位数反而更差，而且两边 `B/op`、`allocs/op` 都没变。这类结果说明收益只停留在噪声边缘，不足以支撑保留代码复杂度。

判断标准不是“均值有一点点优势”就算赢，而是要看：

1. 同一实现是否在关联 benchmark 上同时成立。
2. 中位数是否也支持这个结论，而不是只有均值被几次抖动拉动。
3. 是否真的换来了更少分配或更明确的热点消除。

如果三条里有一条站不住，就把这种“预分配 + 手写 copy”视为未证实优化，直接回退。

# 契约明确的专用快路径可以单独成立

同样是“预分配 + 手写 `copy`”，只有在输入契约比通用路径更强、能明确删掉一层通用分支时，才值得单独保留。

这次在 `internal/gitops/service.go` 里，`IgnoredPathSetFromRootSorted` 的调用方是 `internal/diff/service.go`，传入的是 `ListSyncableSourcePathsFromRoot` 产出的 `sourceFiles`。这条路径的前提比普通 `buildSingleGroupInputWithoutDedup` 更强：

1. 输入已排序。
2. 输入已去重。
3. 输入里没有空串。

在这个前提下，可以把它从通用构造函数里拆出来，走专用 `buildSortedSingleGroupInput`：

1. 先按 `estimateSingleGroupBytes` 一次性定长。
2. 直接按偏移量 `copy` 路径，再补 `'\n'`。
3. 不再重复做“空串跳过”的通用分支判断。

同 session 10 次候选/基线对照结果：

- `BenchmarkIgnoredPathSetFromRootSorted`
- candidate: `average 49909.7 ns/op`, `median 51006.5 ns/op`
- baseline: `average 50258.7 ns/op`, `median 53399.5 ns/op`
- `BenchmarkCalculateLargeDiff`
- candidate: `average 175570.7 ns/op`, `median 182965.5 ns/op`
- baseline: `average 179745.0 ns/op`, `median 184280.0 ns/op`

这里能保留，不是因为“手写 `copy` 天生更快”，而是因为：

1. 优化绑定在一个更强的输入契约上，没有污染通用路径。
2. 目标 benchmark 和真实消费者 benchmark 同时成立。
3. `B/op`、`allocs/op` 虽未变化，但 CPU 路径收益在同会话对照里可重复。

结论是：同一种微优化，放在通用路径上可能只是噪声，放在契约更强的专用入口上才可能形成净收益。

# 计数基准要贴近数据分布改写

如果一个 helper 的职责只是为后续切片预分配算容量，而输入分布本身明显偏向一侧，可以把“逐步累加 union 大小”的写法改成“从更大的稳定基数出发，只在少数分支补增量”。

这次在 `internal/diff/service.go` 里，`mergedPathCount` 的输入来自：

1. `sourceFiles`：完整的 source 可同步文件集。
2. `targetFiles`：通常是 source 的子集或近似子集。

原实现每轮比较都会 `count++`，最后再补 `len(sourceFiles)-sourceIndex` 和 `len(targetFiles)-targetIndex`。在 benchmark 数据里，`targetFiles` 是“每 4 个缺 1 个”的形状，这意味着：

- 绝大多数轮次只是确认 source 已存在的路径；
- 真正需要额外增加 union 大小的，只是 `target` 独有路径分支。

因此可以改成：

1. 先用 `len(sourceFiles)` 作为基数。
2. 只有命中 `sourcePath > targetPath` 时才 `count++`。
3. 循环结束后只补剩余的 `target` 尾段。

同 session 5 次候选/基线对照结果：

- `BenchmarkMergedPathCount`
- candidate: `average 16038.4 ns/op`, `median 16177 ns/op`
- baseline: `average 16346.8 ns/op`, `median 16446 ns/op`
- `BenchmarkCalculateLargeDiff`
- candidate: `average 166800.2 ns/op`, `median 184135 ns/op`
- baseline: `average 169364.4 ns/op`, `median 188567 ns/op`

这个优化能保留的关键，不是“少一次加法”这么简单，而是它顺着真实数据分布改写了计数逻辑：把最常见路径变成默认成本，把少见分支留给增量处理。
