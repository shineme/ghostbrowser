# 开源内核（chromium）版本兼容矩阵

测试日期 2026-10-14，分支 verify-compat。环境：Linux **arm64** 沙箱，headless=new。
Chrome for Testing 只发布 x86_64 Linux 包且下载域名被沙箱拦截，所以用 Debian snapshot 的 arm64 Chromium 各大版本（+ ungoogled-chromium 155 portable）代替；
品牌版 Google Chrome 无法在本环境实测。

| 版本 | 来源 | 指纹仿真 UA-CH/屏幕/触摸 | GPU 字符串 | Canvas 噪声 | WebRTC/WebGPU/媒体设备/语音/硬件/DNT | 端口扫描 | 端口扫描拦截 | 扩展加载 | Cookie | RPA | 窗口排列 | 同步器 | 同步器主控关闭 | UA 大版本/品牌/请求头 + 新标签页注入 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 110.0.5481.177 | Debian | ✅ | n.a.¹ | ✅² | ✅² | ✅ | ✅ | ✅ | ✅ | ⚠️³ | ✅ | ❌⁴ | ✅ | ✅² |
| 115.0.5790.170 | Debian | ✅ | n.a.¹ | ✅² | ✅² | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌⁴ | ✅ | ✅² |
| 120.0.6099.224 | Debian | ✅ | n.a.¹ | ✅ | ✅⁵ | ✅ | ✅ | ✅ | ✅ | ✅⁶ | ✅ | ✅ | ✅ | ✅ |
| 125.0.6422.141 | Debian | ✅ | n.a.¹ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 130.0.6723.116 | Debian | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 135.0.7049.95 | Debian | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 140.0.7339.207 | Debian | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 145.0.7632.159 | Debian | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 150.0.7871.181 | Debian | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 155.0.8059.39 | apt chromium | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 155.0.8059.39 | ungoogled-chromium | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 137+ 品牌版 Chrome | — | 未测（预期 ✅） | | | | | | ❌⁷ | | | | | | |
| Chrome for Testing | — | 未测（预期同 Chromium） | | | | | | ✅⁷ | | | | | | |

1. Debian 的 arm64 Chromium ≤125 没有打包 SwiftShader，无 GPU 的 headless 环境里根本没有 WebGL，无可伪装；不是 GhostBrowser 的问题（测试已改为此时跳过 GPU 检查）。
2. 修复前失败：Chrome <120 里 `addScriptToEvaluateOnNewDocument` 对"带网址新建、被 auto-attach 暂停"的标签页的首个文档不生效（`runImmediately` 也被忽略），整段指纹脚本缺失。现在 <120 自动检测并重载一次。
3. 110 headless 的焦点语义：新驱动接管时选中了拥有焦点的旧标签页而不是最新的；headful 下不受影响，未修。
4. 未解决：<120 的同步器捕获脚本在隔离世界里已安装，但真实点击没有进入队列（单独探针里同样的代码可以工作）。超过两轮排查，按要求停止。同步器建议 120+。
5. 修复前失败：Chromium 120 没有暴露 `SpeechSynthesisVoice` 接口，自定义语音列表为空；现在用私有原型兜底。
6. 修复前失败：内核刚开调试端口时 110–120 还没有注册首个标签页，RPA 驱动会多开一个 about:blank；现在先等待最多 3 秒。
7. 依据 Google 公告（https://groups.google.com/a/chromium.org/g/chromium-extensions/c/1-g8EFx2BBY）：137 起品牌版 Chrome 忽略 `--load-extension`，Chromium / Chrome for Testing 不受影响；`DisableLoadExtensionCommandLineSwitch` 规避据报告在 142 起失效。

并发跑 3 个内核时偶发 "debugger not ready" / "Execution context was destroyed"（负载导致），这些用例单独重跑全部通过。

## 推荐
- 最低支持：**110**（低于 110 拒绝启动并给出中文错误）。110–119 可用但会提示升级（新标签页多加载一次；同步器不可用）。
- 推荐：**120 及以上**的 Chromium / ungoogled-chromium / Chrome for Testing；最佳为最新稳定版（155）。
- 需要扩展时不要用品牌版 Chrome 137+；Edge 的 UA-CH 品牌为 Microsoft Edge，与 Chrome 模板不一致。
