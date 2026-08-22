# whisper-gui

[English](#english) | [中文](#中文)

A local, **macOS-only** web GUI wrapping [`mlx_whisper`](https://github.com/ml-explore/mlx-examples/tree/main/whisper) for batch video transcription to `.srt` subtitle files. A Go backend serves a small vanilla TypeScript/Vite frontend (no framework) and drives `mlx_whisper`/`ffmpeg` as subprocesses.

---

## English

### Why macOS-only

`mlx_whisper` is built on Apple's [MLX](https://github.com/ml-explore/mlx) framework, which targets Apple Silicon's unified-memory architecture. It does not run on Intel Macs, Windows, or Linux. The app's config/job-history storage path (`~/Library/Application Support/whisper-gui/`) is also macOS-specific.

### Features

- Server-side file browser to pick one or more video files (checkbox multi-select, video files only).
- Pick a Whisper model and language, then queue transcription jobs.
- Jobs run **one at a time** — MLX's unified-memory GPU path gains nothing from parallel jobs on a single Mac, so a single-worker FIFO queue avoids GPU memory contention.
- Live job progress and logs via Server-Sent Events (auto-reconnect, log replay after a server restart).
- Cancel a running or queued job.
- Download the resulting `.srt`; a copy is also saved next to the source video (best-effort).
- Settings page to override the `mlx_whisper`/`ffmpeg` binary paths if auto-detection fails.

### Requirements

- macOS on Apple Silicon (M1/M2/M3/M4).
- [`mlx_whisper`](https://pypi.org/project/mlx-whisper/) installed via pip.
- [`ffmpeg`](https://ffmpeg.org/) installed, e.g. via Homebrew (`brew install ffmpeg`).
- Go 1.26+ and Node.js (for building from source).

Both binaries are auto-detected on `PATH` (with a pip user-install fallback for `mlx_whisper`); if detection fails, set explicit paths in the app's Settings page.

### Build & run

```bash
make build   # builds the frontend into internal/server/dist, then builds the Go binary
make run     # build + run ./bin/whisper-gui
```

Other targets:

```bash
make dev-api   # go run ./cmd/whisper-gui — starts the API server on :8787
make dev-web   # cd web && npm run dev — Vite dev server, proxies /api to :8787
make test      # go test ./...
make clean     # remove build artifacts (bin/, web/node_modules, web/dist, internal/server/dist)
```

For frontend-only development, run `make dev-api` and `make dev-web` side by side, then open the Vite dev server URL.

Once built, `whisper-gui` serves everything (API + embedded frontend) on a single port:

```bash
./bin/whisper-gui -port 8787
```

### Data storage

All job history, subprocess logs, and generated `.srt` files live under `~/Library/Application Support/whisper-gui/` (`config.json`, `jobs.json`, `jobs/<id>/`) — nothing is written into the repo. The `srt/` folder in this repo, if present locally, is just a manually-curated copy pulled from there and is gitignored.

### Project structure

- `cmd/whisper-gui` — binary entrypoint; flag parsing, startup wiring, graceful shutdown.
- `internal/jobmanager` — job state machine, single-worker FIFO transcription queue, subprocess execution, SSE event fan-out.
- `internal/server` — HTTP handlers (standard library `net/http`), embeds the built frontend via `go:embed`.
- `internal/config` — JSON config file for user-overridden binary paths.
- `internal/whisperbin` — resolves `mlx_whisper`/`ffmpeg` binary paths.
- `web/` — vanilla TypeScript + Vite frontend, built straight into `internal/server/dist`.

See the `CLAUDE.md` file in each directory for implementation details.

---

## 中文

### 为什么只支持 macOS

`mlx_whisper` 基于苹果的 [MLX](https://github.com/ml-explore/mlx) 框架构建，该框架针对 Apple Silicon 的统一内存架构设计，无法在 Intel Mac、Windows 或 Linux 上运行。此外，本应用的配置和任务历史存储路径（`~/Library/Application Support/whisper-gui/`）也是 macOS 专属路径。

### 功能特性

- 服务端文件浏览器，可勾选一个或多个视频文件（多选，仅视频文件可选）。
- 选择 Whisper 模型和语言，将转写任务加入队列。
- 任务**串行执行**——MLX 的统一内存 GPU 架构在单台 Mac 上并行跑多个任务并不能带来收益，单 worker 的 FIFO 队列可以避免 GPU 内存争用。
- 通过 Server-Sent Events 实时展示任务进度与日志（自动重连，服务重启后可回放历史日志）。
- 支持取消排队中或正在运行的任务。
- 转写完成后可下载 `.srt` 文件，同时会尽力在源视频旁保存一份副本。
- 设置页面可在自动检测失败时手动指定 `mlx_whisper`/`ffmpeg` 的可执行文件路径。

### 环境要求

- 搭载 Apple Silicon（M1/M2/M3/M4）芯片的 macOS。
- 通过 pip 安装的 [`mlx_whisper`](https://pypi.org/project/mlx-whisper/)。
- 已安装 [`ffmpeg`](https://ffmpeg.org/)，例如通过 Homebrew 安装（`brew install ffmpeg`）。
- 若需从源码构建：Go 1.26+ 和 Node.js。

两个可执行文件默认会从 `PATH` 中自动检测（`mlx_whisper` 还有 pip 用户安装路径作为兜底）；如果自动检测失败，可在应用的设置页面手动指定路径。

### 构建与运行

```bash
make build   # 先将前端构建到 internal/server/dist，再构建 Go 二进制
make run     # 构建并运行 ./bin/whisper-gui
```

其他常用命令：

```bash
make dev-api   # go run ./cmd/whisper-gui —— 在 :8787 启动 API 服务
make dev-web   # cd web && npm run dev —— 启动 Vite 开发服务器，将 /api 代理到 :8787
make test      # go test ./...
make clean     # 清理构建产物（bin/、web/node_modules、web/dist、internal/server/dist）
```

如果只想开发前端，可以同时运行 `make dev-api` 和 `make dev-web`，然后打开 Vite 开发服务器给出的地址。

构建完成后，`whisper-gui` 会用同一个端口同时提供 API 和前端页面：

```bash
./bin/whisper-gui -port 8787
```

### 数据存储

所有任务历史、子进程日志和生成的 `.srt` 文件都保存在 `~/Library/Application Support/whisper-gui/`（`config.json`、`jobs.json`、`jobs/<id>/`）——不会写入仓库目录。仓库中的 `srt/` 目录（如果本地存在）只是从该目录手动拉取的副本，已被 gitignore 忽略。

### 项目结构

- `cmd/whisper-gui` —— 二进制入口：命令行参数解析、启动装配、优雅关闭。
- `internal/jobmanager` —— 任务状态机、单 worker 的 FIFO 转写队列、子进程执行、SSE 事件分发。
- `internal/server` —— HTTP 处理层（标准库 `net/http`），通过 `go:embed` 内嵌构建好的前端。
- `internal/config` —— JSON 配置文件，保存用户自定义的可执行文件路径。
- `internal/whisperbin` —— 负责解析 `mlx_whisper`/`ffmpeg` 的可执行文件路径。
- `web/` —— 原生 TypeScript + Vite 前端，直接构建到 `internal/server/dist`。

更多实现细节参见各目录下的 `CLAUDE.md` 文件。
