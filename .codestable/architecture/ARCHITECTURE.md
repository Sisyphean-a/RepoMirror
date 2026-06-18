# RepoMirror 架构总入口

> 状态：current
> 更新日期：2026-06-18

## 1. 项目简介

RepoMirror 是一个用 Go 和 Wails 实现的 Windows 桌面工具，用于在两个 Git 仓库目录之间执行双向镜像同步。

首版交付聚焦三个核心面：

- 本地配置：记住仓库 A/B 路径、同步方向、窗口尺寸
- 差异计算：用源端可同步文件集合和目标端 ignore 保护规则共同生成差异列表
- 目标仓库动作：查看 Git 状态，并对目标仓库执行同步、提交、推送

## 2. 核心概念 / 术语表

- `Direction`：当前同步方向，值为 `A_TO_B` 或 `B_TO_A`
- `SourceRoot`：当前方向下的源仓库根目录
- `TargetRoot`：当前方向下的目标仓库根目录
- `DiffEntry`：单个差异文件，类型为 `added` / `modified` / `deleted`
- `TargetRepositoryStatus`：目标仓库分支、是否干净、未提交数量、未跟踪数量

## 3. 子系统 / 模块索引

- `main.go`：组装依赖，创建 Wails 应用
- `app.go` + `internal/app/`：Wails 绑定层与应用服务编排
- `internal/config/`：配置文件读写，路径默认 `%APPDATA%/RepoMirror/config.json`
- `internal/gitops/`：`git rev-parse` / `ls-files` / `check-ignore` / `status` / `branch` / `commit` / `push`
- `internal/diff/`：基于源端文件集和目标端 ignore 保护规则计算差异
- `internal/syncer/`：按差异结果复制或删除目标仓库文件
- `internal/platform/`：文件系统抽象，供 diff / syncer 复用
- `frontend/src/`：React UI，主视图由控制区、差异区、目标状态区组成

## 4. 关键架构决定

- Git 交互统一通过本机 `git` 命令完成，不引入额外 Git SDK
- 差异计算不依赖时间戳；若文件同时存在则以实际字节比较判断是否修改
- 源仓库输出集合采用 `git ls-files --cached --others --exclude-standard -z`
- 目标仓库保护集合由三部分组成：`.git`、任意层级 `.gitignore`、`git check-ignore` 命中的路径
- 前端不直接拼装业务数据，只消费后端返回的 `DashboardState`

## 5. 已知约束 / 硬边界

- `.git` 目录内容绝不复制、删除或覆盖
- `.gitignore` 文件绝不进入差异列表，也不参与同步
- 目标仓库已有未提交改动时仍允许同步，但必须显式展示状态，由用户自行决定
- 提交与推送操作只作用于当前目标仓库，不影响源仓库
- 当前验证覆盖了配置持久化、差异计算、同步执行；桌面 UI 主要通过 `npm run build` 与 `wails build` 保证可构建
