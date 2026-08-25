# whisper-gui

本地 macOS 专用的网页 GUI，封装 `mlx_whisper`，用于批量把视频转写成 `.srt` 字幕。Go 后端 + vanilla TS/Vite 前端（无框架），驱动 `mlx_whisper`/`ffmpeg` 子进程完成转写。

## 前置依赖

- **mlx_whisper**（pip 安装，需要 Apple Silicon）
- **ffmpeg**（`brew install ffmpeg`）

两者由 `internal/whisperbin` 在运行时自动探测路径（config 覆盖 → `PATH` → 已知 pip 安装位置）；探测失败时可在 UI 的 Settings 里手动指定路径。

## 运行

开发模式（前后端分开跑，两个终端，前端支持热重载）：

```bash
make dev-api   # 后端，监听 :8787
make dev-web   # 前端 Vite dev server，代理 /api 到 :8787
```

或者一条命令同时启动（同一终端交织输出，Ctrl+C 一起退出）：

```bash
make dev
```

生产/单进程模式（前端会先重新构建，再编译打包进 Go 二进制）：

```bash
make run
```

访问 `http://localhost:8787`（`make dev-web` 模式下访问 Vite 给出的地址，通常是 `http://localhost:5173`）。

其他：

```bash
make build   # 只构建，不运行
make test    # go test ./...
```

## 支持的模型

| 模型 | 参数量 | 速度 | 准确率 | 说明 |
|---|---|---|---|---|
| medium | ~769M | 较慢 | 较准 | 速度与准确率的折中，清晰单人语音场景够用 |
| large-v3-turbo | ~809M | 快 | 接近 large-v3 | large-v3 的剪枝加速版，日常首选 |
| large-v3 | ~1.5B | 最慢 | 最准 | 背景噪音多/口音重/多人交叉场景更值得用 |

模型需要提前通过 `mlx_whisper`（HuggingFace Hub）下载好；App 会读取本地 HuggingFace 缓存（`~/.cache/huggingface/hub/`）判断模型是否已就绪，未下载的模型会在 UI 里标注「needs download」。

## 运行时状态（不在 git 里）

任务历史、子进程输出日志、生成的 `.srt` 都保存在 `~/Library/Application Support/whisper-gui/`（`config.json`、`jobs.json`、`jobs/<id>/`），不进仓库。仓库里的 `srt/` 目录只是从那里手动挑出来的字幕副本，与 App 本身无关。

## 更多细节

架构、目录职责、开发约定见 [CLAUDE.md](CLAUDE.md)，每个子目录下也各自有一份 `CLAUDE.md`。
