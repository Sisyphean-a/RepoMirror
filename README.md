# RepoMirror

`RepoMirror` 是一个基于 `Go + Wails + React` 的 Windows 桌面工具，用于在两个 Git 仓库目录之间执行双向镜像同步。

## 当前能力

- 选择并持久化仓库 A / 仓库 B 路径
- 切换 `A → B` / `B → A` 同步方向
- 展示差异文件列表，并标记 `新增` / `修改` / `删除`
- 同步时同时遵守源端输出规则和目标端 ignore 保护规则
- 展示目标仓库当前分支、未提交数量、未跟踪数量、是否干净
- 对目标仓库执行 `git add -A` + `git commit -m` + `git push`

## 硬边界

- `.git` 永不修改
- 任意层级 `.gitignore` 永不参与同步
- 被目标仓库 ignore 规则命中的文件不会被覆盖或删除
- 不实现 merge / rebase / stash / 冲突解决等复杂 Git 工作流

## 开发

安装依赖后可直接运行：

```bash
wails dev
```

前端单独构建：

```bash
cd frontend
npm run build
```

后端测试：

```bash
go test ./...
```

## 构建

生成桌面可执行文件：

```bash
wails build
```

当前构建产物位置：

- `build/bin/RepoMirror.exe`
