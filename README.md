# whisper-gui

[English](#english) | [中文](#中文)

A local, **macOS-only** web GUI for batch video transcription to `.srt` subtitle files. A Go backend serves a small vanilla TypeScript/Vite frontend (no framework) and drives one of two transcription engines as a subprocess: [`mlx_whisper`](https://github.com/ml-explore/mlx-examples/tree/main/whisper) (Python/MLX) or [whisper.cpp](https://github.com/ggml-org/whisper.cpp)'s `whisper-cli` (pure C/C++, no Python dependency at all). `ffmpeg` is used by both.

---

## English

### Why macOS-only

Both engines target Apple Silicon: `mlx_whisper` is built on Apple's [MLX](https://github.com/ml-explore/mlx) framework, and whisper.cpp is typically installed via a Homebrew formula built with Metal acceleration. The app's config/job-history storage path (`~/Library/Application Support/whisper-gui/`) is also macOS-specific.

### Features

- Server-side file browser to pick one or more video files (checkbox multi-select, video files only).
- Pick an engine (`mlx_whisper` or whisper.cpp), a model, and a language, then queue transcription jobs.
- Jobs run **one at a time**, regardless of engine — a single-worker FIFO queue avoids GPU memory contention (MLX's unified-memory GPU path gains nothing from parallel jobs on a single Mac).
- Live job progress and logs via Server-Sent Events (auto-reconnect, log replay after a server restart).
- Cancel a running or queued job.
- Download the resulting `.srt`; a copy is also saved next to the source video (best-effort).
- Settings panel to override the `mlx_whisper`/`ffmpeg`/`whisper-cli` binary paths if auto-detection fails, set the whisper.cpp model directory, and pick a default engine.

### Requirements

- macOS on Apple Silicon (M1/M2/M3/M4).
- At least one transcription engine:
  - [`mlx_whisper`](https://pypi.org/project/mlx-whisper/) installed via pip, **or**
  - whisper.cpp's `whisper-cli` installed via Homebrew (`brew install whisper-cpp`) — no Python/pip setup needed at all, which makes it the simpler choice on a freshly set-up Mac. Model files (`ggml-*.bin`) must be downloaded manually into a directory of your choice, then pointed at from Settings; there is no auto-download for this engine.
- [`ffmpeg`](https://ffmpeg.org/) installed, e.g. via Homebrew (`brew install ffmpeg`) — required by both engines (whisper.cpp uses it directly to convert video into 16kHz mono WAV before transcribing; mlx_whisper calls it internally).
- Go 1.26+ and [Bun](https://bun.sh) (for building from source).

All three binaries are auto-detected on `PATH` (with a pip user-install fallback for `mlx_whisper` only); if detection fails, set explicit paths in the app's Settings panel. If `mlx_whisper` isn't installed but `whisper-cli` is, the app automatically defaults to the whisper.cpp engine on first load (unless you've explicitly picked a default engine in Settings).

> **Silence-triggered repeated/hallucinated text**: mlx_whisper's version of this (see `notes/mlx-whisper-repetition-loop-bugfix.md`) is fixed via `--condition-on-previous-text False`. whisper.cpp has no direct equivalent of that flag, but it does support **Voice Activity Detection (VAD)**, which skips silent stretches before they ever reach the decoder — the actual fix for this failure mode. It's optional: download a ggml VAD model (e.g. `ggml-silero-v5.1.2.bin` from [`ggml-org/whisper-vad`](https://huggingface.co/ggml-org/whisper-vad) on Hugging Face) and set its path in the Settings panel's "whisper.cpp VAD model" field. Leave it unset and whisper.cpp runs exactly as before (no behavior change), but a silence-heavy recording is then worth spot-checking for repeated text.

### Build & run

```bash
make build   # builds the frontend into internal/server/dist, then builds the Go binary
make run     # build + run ./bin/whisper-gui
```

Other targets:

```bash
make dev-api   # go run ./cmd/whisper-gui — starts the API server on :8787
make dev-web   # cd web && bun run dev — Vite dev server, proxies /api to :8787
make test      # go test ./...
make clean     # remove build artifacts (bin/, web/node_modules, web/dist, internal/server/dist, tmp)
```

For frontend-only development, run `make dev-api` and `make dev-web` side by side, then open the Vite dev server URL.

For backend auto-reload (rebuild + restart on every saved `.go` file), use [`air`](https://github.com/air-verse/air):

```bash
go install github.com/air-verse/air@latest   # one-time install

make dev-api-watch   # like dev-api, but auto-rebuilds/restarts on .go save
make dev-watch        # dev-api-watch + dev-web together
```

> `dev-api-watch` restarts the server on every saved `.go` file — don't use it while a transcription job you care about is running, it will be killed just like a manual Ctrl+C would.

Once built, `whisper-gui` serves everything (API + embedded frontend) on a single port:

```bash
./bin/whisper-gui -port 8787
```

### Data storage

All job history, subprocess logs, and generated `.srt` files live under `~/Library/Application Support/whisper-gui/` (`config.json`, `jobs.json`, `jobs/<id>/`) — nothing is written into the repo. The `srt/` folder in this repo, if present locally, is just a manually-curated copy pulled from there and is gitignored.

### Project structure

- `cmd/whisper-gui` — binary entrypoint; flag parsing, startup wiring, graceful shutdown.
- `internal/jobmanager` — job state machine, single-worker FIFO transcription queue, the `Engine` abstraction over `mlx_whisper`/whisper.cpp, subprocess execution, SSE event fan-out.
- `internal/server` — HTTP handlers (standard library `net/http`), embeds the built frontend via `go:embed`.
- `internal/config` — JSON config file for user-overridden binary paths, whisper.cpp model directory, and default engine.
- `internal/whisperbin` — resolves `mlx_whisper`/`ffmpeg`/`whisper-cli` binary paths.
- `web/` — vanilla TypeScript + Vite frontend, built straight into `internal/server/dist`.

See the `CLAUDE.md` file in each directory for implementation details.

---

## 中文

### 为什么只支持 macOS

两个引擎都是面向 Apple Silicon 设计的：`mlx_whisper` 基于苹果的 [MLX](https://github.com/ml-explore/mlx) 框架构建；whisper.cpp 通常通过 Homebrew 安装，构建时启用了 Metal 加速。此外，本应用的配置和任务历史存储路径（`~/Library/Application Support/whisper-gui/`）也是 macOS 专属路径。

### 功能特性

- 服务端文件浏览器，可勾选一个或多个视频文件（多选，仅视频文件可选）。
- 选择引擎（`mlx_whisper` 或 whisper.cpp）、模型和语言，将转写任务加入队列。
- 任务**串行执行**，与引擎无关——单 worker 的 FIFO 队列可以避免 GPU 内存争用（MLX 的统一内存 GPU 架构在单台 Mac 上并行跑多个任务并不能带来收益）。
- 通过 Server-Sent Events 实时展示任务进度与日志（自动重连，服务重启后可回放历史日志）。
- 支持取消排队中或正在运行的任务。
- 转写完成后可下载 `.srt` 文件，同时会尽力在源视频旁保存一份副本。
- 设置面板可在自动检测失败时手动指定 `mlx_whisper`/`ffmpeg`/`whisper-cli` 的可执行文件路径，设置 whisper.cpp 模型目录，并选择默认引擎。

### 环境要求

- 搭载 Apple Silicon（M1/M2/M3/M4）芯片的 macOS。
- 至少安装一个转录引擎：
  - 通过 pip 安装的 [`mlx_whisper`](https://pypi.org/project/mlx-whisper/)，**或者**
  - 通过 Homebrew 安装的 whisper.cpp 的 `whisper-cli`（`brew install whisper-cpp`）——完全不需要搭建 Python/pip 环境，在全新安装的 Mac 上是更简单的选择。模型文件（`ggml-*.bin`）需要手动下载到自选目录，再到设置里指定该目录；这个引擎没有自动下载功能。
- 已安装 [`ffmpeg`](https://ffmpeg.org/)，例如通过 Homebrew 安装（`brew install ffmpeg`）——两个引擎都需要它（whisper.cpp 会直接调用它把视频转成 16kHz 单声道 WAV 再转录；mlx_whisper 则是内部自行调用）。
- 若需从源码构建：Go 1.26+ 和 [Bun](https://bun.sh)。

三个可执行文件默认都会从 `PATH` 中自动检测（仅 `mlx_whisper` 还有 pip 用户安装路径作为兜底）；如果自动检测失败，可在应用的设置面板手动指定路径。如果没有安装 `mlx_whisper` 但装了 `whisper-cli`，应用首次打开时会自动默认选中 whisper.cpp 引擎（除非你已经在设置里明确指定了默认引擎）。

> **静音触发复读/幻觉文本**：mlx_whisper 那个问题（详见 `notes/mlx-whisper-repetition-loop-bugfix.md`）是靠 `--condition-on-previous-text False` 修复的。whisper.cpp 没有直接对应这个选项的参数，但它支持 **VAD（语音活动检测）**——在音频进解码器之前就先跳过静音片段，这才是这类问题真正对症的修复方式。这项是可选的：下载一个 ggml 格式的 VAD 模型（例如 Hugging Face 上 [`ggml-org/whisper-vad`](https://huggingface.co/ggml-org/whisper-vad) 的 `ggml-silero-v5.1.2.bin`），然后在设置面板的 "whisper.cpp VAD model" 里填上路径。不填的话 whisper.cpp 行为跟之前完全一样(不影响现有配置)，但转录静音较多的录音时就值得抽查一下有没有复读。

### 构建与运行

```bash
make build   # 先将前端构建到 internal/server/dist，再构建 Go 二进制
make run     # 构建并运行 ./bin/whisper-gui
```

其他常用命令：

```bash
make dev-api   # go run ./cmd/whisper-gui —— 在 :8787 启动 API 服务
make dev-web   # cd web && bun run dev —— 启动 Vite 开发服务器，将 /api 代理到 :8787
make test      # go test ./...
make clean     # 清理构建产物（bin/、web/node_modules、web/dist、internal/server/dist、tmp）
```

如果只想开发前端，可以同时运行 `make dev-api` 和 `make dev-web`，然后打开 Vite 开发服务器给出的地址。

如果想要后端热更新（保存 `.go` 文件后自动重新编译并重启），可以用 [`air`](https://github.com/air-verse/air)：

```bash
go install github.com/air-verse/air@latest   # 一次性安装

make dev-api-watch   # 类似 dev-api，但保存 .go 文件后会自动重新编译并重启
make dev-watch        # dev-api-watch + dev-web 一起启动
```

> `dev-api-watch` 会在每次保存 `.go` 文件时重启服务——如果这时候有你在意的转写任务正在跑，不要用它，效果和手动 Ctrl+C 一样会把任务杀掉。

构建完成后，`whisper-gui` 会用同一个端口同时提供 API 和前端页面：

```bash
./bin/whisper-gui -port 8787
```

### 数据存储

所有任务历史、子进程日志和生成的 `.srt` 文件都保存在 `~/Library/Application Support/whisper-gui/`（`config.json`、`jobs.json`、`jobs/<id>/`）——不会写入仓库目录。仓库中的 `srt/` 目录（如果本地存在）只是从该目录手动拉取的副本，已被 gitignore 忽略。

### 项目结构

- `cmd/whisper-gui` —— 二进制入口：命令行参数解析、启动装配、优雅关闭。
- `internal/jobmanager` —— 任务状态机、单 worker 的 FIFO 转写队列、`mlx_whisper`/whisper.cpp 的 `Engine` 抽象、子进程执行、SSE 事件分发。
- `internal/server` —— HTTP 处理层（标准库 `net/http`），通过 `go:embed` 内嵌构建好的前端。
- `internal/config` —— JSON 配置文件，保存用户自定义的可执行文件路径、whisper.cpp 模型目录、默认引擎。
- `internal/whisperbin` —— 负责解析 `mlx_whisper`/`ffmpeg`/`whisper-cli` 的可执行文件路径。
- `web/` —— 原生 TypeScript + Vite 前端，直接构建到 `internal/server/dist`。

更多实现细节参见各目录下的 `CLAUDE.md` 文件。
