# Go 基础命令笔记

> 学习背景：从 Python/Java 转过来看 whisper-gui 这个项目时整理的笔记，定期复习用。

## 四个核心命令

| 命令 | 作用 | 用在什么场景 |
|---|---|---|
| `go run ./cmd/xxx` | 临时编译 + 立即执行，不留下二进制文件（产物在系统临时目录，跑完就扔） | 日常开发迭代，改一行代码重跑一次 |
| `go build -o bin/xxx ./cmd/xxx` | 只编译，产出**持久化**的可执行文件，不执行 | 打包分发；或者像本项目一样，先 build 前端再 `go:embed` 进最终二进制 |
| `go test ./...` | 真的运行测试——执行 `_test.go` 里的 `TestXxx` 函数，验证代码**行为**对不对 | 提交前 / CI |
| `go vet ./...` | 静态分析，抓"编译能过、但语义上大概率是 bug"的写法（不是语法检查，语法错误编译器自己就直接拒绝了） | 提交前 / CI，常见 vet 案例：`Printf` 格式符和参数类型对不上、struct tag 写错、`sync.Mutex` 被值拷贝、goroutine 闭包捕获循环变量 |

`vet` 这个词本身是"审查、甄别"的意思（e.g. "vet a job candidate"），Go 团队取这个名字就是要表达"审查代码里可疑的地方"。

## `go:embed` 是什么

Go 1.16+ 的编译期指令，能把静态文件（HTML/CSS/JS、图片、模板等）**直接打包进编译出的二进制文件**，运行时不再依赖磁盘上这些文件是否存在。

本项目里的用法（[internal/server/embed.go](../internal/server/embed.go)）：

```go
//go:embed dist
var distFS embed.FS
```

`make build` 先把前端 Vite 构建产物放到 `internal/server/dist/`，`go build` 编译时把整个 `dist/` 目录内容"塞进"最终的 `bin/whisper-gui` 里。好处：分发/运行只需要**一个可执行文件**，不用额外带着前端静态文件到处拷贝。

## 依赖管理：为什么不用像 Python 那样纠结 venv/poetry

**Python 的痛点**：两个项目依赖同一个库的不同版本（比如 `requests==2.20` vs `requests==2.31`），如果都装在系统全局会互相覆盖打架。所以每个项目要建一个隔离的虚拟环境（`venv`），进入项目前要「激活」对应的环境（`source .venv/bin/activate`），切项目要记得切环境——忘了 activate 或 activate 错环境，是常见踩坑点。

**Go 怎么绕开这个问题**：依赖版本不是靠"当前激活了哪个环境"来区分，而是**每个项目自己在代码里写死一份清单**：

- `go.mod` —— 这个项目依赖谁、依赖哪个版本
- `go.sum` —— 每个依赖包内容的哈希校验值，防篡改

`go run`/`go build` 执行时会：
1. 读本项目的 `go.mod`，确定需要哪个包的哪个版本
2. 去共享缓存 `$GOPATH/pkg/mod/<包名>@<版本号>/` 找，有就直接用，没有就下载
3. 缓存是按"包名+版本号"分文件夹的，`v1.2.3` 和 `v2.0.0` 是两个完全不同的文件夹，跨项目共享但互不覆盖

所以不同项目各自读自己的 `go.mod`，各自从这个共享缓存里取对应版本，天然不会打架，也就不需要「先弄清楚我现在处于哪个隔离环境」这种心智负担——克隆仓库后直接在项目目录里 `go run .` 就能跑，因为**项目文件夹本身 + go.mod/go.sum 就包含了跑起来所需的全部信息**，不依赖任何外部激活状态。

（本项目 [go.mod](../go.mod) 目前只有模块名和 Go 版本号，没有第三方依赖——项目约定"Go 标准库 only"，所以看不到 `require` 这一行。）

## 本项目里几个命令的关系（回顾）

```
make dev-api  →  go run ./cmd/whisper-gui         （后端，热重跑，不留二进制）
make dev-web  →  cd web && npm run dev             （前端 Vite dev server，代理 /api）
make dev      →  上面两个一起跑（trap 'kill 0' EXIT 保证 Ctrl+C 一起退出）
make build    →  npm run build + go build -o bin/whisper-gui  （前端先打包，再嵌入 Go 二进制）
make run      →  依赖 build，然后执行 ./bin/whisper-gui        （单进程验证最终打包效果）
```
