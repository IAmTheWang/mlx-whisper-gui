# mlx_whisper 重复识别（repetition loop）排查记录

> 排查日期：2026-09-04。起因：用户反馈"最近一周字幕识别准确率暴跌，出现大量重复且错误的识别内容，字幕无法使用"，怀疑是不是 bug 或者需要重启项目。

## 症状

`~/Library/Application Support/whisper-gui/jobs/<job-id>/output.log` 里能看到类似这样的输出，一直持续到文件结尾：

```
[stdout] [17:44.000 --> 17:46.000] お待ちしております
[stdout] [17:46.000 --> 17:48.000] お待ちしております
[stdout] [17:48.000 --> 17:50.000] お待ちしております
...（一路重复到 19 分钟结尾）
```

对应的 `.srt` 从某一句开始，同一句话被重复上百次，subtitle 彻底不可用。

## 排查过程

### 第一步：排除"最近代码/环境变了"

- `git log`：仓库最近一次 commit 是 8/25，排查期间（9/1 之后）没有任何 commit——**代码没变**。
- `pip show mlx-whisper`（0.4.3）、`mlx`（0.29.3）、`brew info ffmpeg`（9.0.1），三者的安装时间都停在 8/17～8/19，**没有升级过**。
- `~/.cache/huggingface/hub/models--mlx-community--whisper-*` 缓存目录的 mtime 也停在 8/17～8/19，**模型文件没有被重新下载或损坏**。

结论：这不是这周"退化"出来的新 bug，环境和代码都是稳定的。

### 第二步：一开始怀疑模型选择（后来证明只是次要问题）

前端 [jobForm.ts](../web/src/views/jobForm.ts) 的模型下拉框当时**没有默认值也不记忆上次选择**——每次刷新页面都会停在 `<select>` 的第一个 `<option>`，也就是 [models.go](../internal/server/models.go) 里 `curatedModels` 列表的第一项 `whisper-large-v3-turbo`（标注为"fast"，比 `large-v3-mlx`"most accurate"更快但没那么准）。

这个问题确实存在，但**不是这次大量重复识别的主因**——因为同一份录音，无论用 turbo 还是 large-v3-mlx（"最准"的模型）跑，都出现了一模一样的复读症状。

### 第三步：真正的根因——mlx_whisper 的 `condition_on_previous_text` 复读循环

看 `mlx_whisper --help`（`/Users/welby/Library/Python/3.9/lib/python/site-packages/mlx_whisper/cli.py`）：

```
--condition-on-previous-text
    If True, provide the previous output of the model as a prompt for the
    next window; disabling may make the text inconsistent across windows,
    but the model becomes less prone to getting stuck in a failure loop
    (default: True)
```

默认 `True`：Whisper 把上一个 30 秒窗口的识别结果，当作下一个窗口解码的"提示词"喂回去。正常语音下这样能让转写更连贯；但只要某一个窗口因为**静音、无声、噪音**而被误判（模型会瞎编一句听起来合理的话，比如日语场景下经典的 "お待ちしております" "ご視聴ありがとうございました"），接下来的每个窗口都会拿着这句错误结果去"提示"自己，从而**卡进一个自我强化的复读循环，一直重复到文件结尾，再也回不来**。

用 `ffmpeg -af silencedetect` 测了触发问题的那条录音（`2026-09-04_11-59-23.mp4`），确认开头几分钟确实有大段静音（比如 34s~100s 静音 66 秒，112s~225s 静音接近 2 分钟——这是"提前开始录制、等人陆续入会"的典型静音模式），这正是触发复读循环的经典条件。

而 [manager.go](../internal/jobmanager/manager.go) 调用 mlx_whisper 时，从项目创建以来就只传了 `--model / --output-dir / --output-name / --output-format / --language`，**从没设置过 `--condition-on-previous-text`**，所以这个坑从第一次调用起就一直存在。

### 第四步：扫全部历史任务，发现这个 bug 其实从第一天就在

写了个脚本扫描 `~/Library/Application Support/whisper-gui/jobs/` 下**全部**历史 `.srt`，统计"连续重复行"最长的一段：

| 任务日期 | 最长连续重复 / 总行数 | 占比 |
|---|---|---|
| 8/17 批次（源视频 8/14 录制） | 2828 / 3889（"はい"） | 73% |
| 8/17 批次（源视频 8/6 录制） | 955 / 1007（"はい"） | 95% |
| 8/19（源视频 8/19 录制） | 644 / 871（"分かりました"） | 74% |
| 8/21 | 2816 / 3004（"はいありがとうございます"） | 94% |
| 8/25 | 525 / 533（"聞こえてると思います"） | **98.5%** |
| 8/27 | 1014 / 1147（"なるほど"） | 88% |
| 9/4（今天，触发本次排查） | 849 / 849（"お待ちしております"） | 100% |

**结论：这不是这周才出现的新 bug，从项目第一次实际调用 mlx_whisper（8/17）起就一直存在**，而且早期部分任务（8/21、8/25）的复读比例甚至比这次更夸张（98.5%）。之前没被当成问题，大概率是因为没有逐字核对过完整字幕，而不是技术上真的没问题。

## 修复

在 [manager.go](../internal/jobmanager/manager.go) 的 `runJob` 里给 mlx_whisper 的调用参数加了一行：

```go
"--condition-on-previous-text", "False",
```

## 验证

用同一条"重灾区"录音（`2026-09-04_11-59-23.mp4`），直接调 mlx_whisper CLI 加上这个参数做对比：

| | 修复前 | 修复后 |
|---|---|---|
| "お待ちしております" 出现次数 | 291（占全部字幕 100%） | 1 |
| 结果 | 全片复读，不可用 | 开头约 4:30 之前是真实静音（会议还没开始），Whisper 照样会脑补几句套话（这是模型固有行为，不算 bug）；但**一旦 4:30 有人说话（"すみません遅れましたお疲れ様です"），立刻恢复正常识别**，后面 14 分钟的会议内容全部转写准确 |

`go build ./...` / `go test ./internal/...` 均通过。

## 后续可选项（未处理，留给用户决定）

1. **重新构建 + 重启服务**：改动只在源码里，需要 `make build` 并重启 `bin/whisper-gui` 才会生效。
2. **补跑今天的两条坏字幕**（`job_1788491928949107000_6`、`job_1788492527264597000_7`），换成修复后的正确版本。
3. **回溯重跑历史上被污染的会议记录**（上面表格列出的那几条，8/21、8/25、8/27 等），如果这些会议内容还有价值，源视频应该还在 `~/Movies/`，可以用修复后的版本重新转写找回来。
4. **前端模型下拉框默认值问题**（[jobForm.ts](../web/src/views/jobForm.ts) 里那个未提交的改动）：目前只是把默认索引指向 `large-v3-mlx`，还没有像 [browser.ts](../web/src/views/browser.ts) 记住 `lastPath` 那样把"上次选的模型"存进 `localStorage`——刷新页面依然会回到默认值。这是独立的小问题，不影响本次修复的结论。

## 参考

- 排查时的代码基线：commit `af8c804`（HEAD，2026-08-25）。
- mlx_whisper 版本：0.4.3；mlx：0.29.3；ffmpeg：9.0.1。
