# SKM macOS 原生 App

当前原生首版为 `0.6.0`，实现技术设计中的 Core 契约与原生 MVP 主流程：

- SwiftUI 三栏导航与 macOS 原生菜单、搜索、文件面板、剪贴板和 Settings scene；
- Skill 本地目录/ZIP/Git/安装命令导入、详情、原文编辑、标签、删除；
- Agent 检测、管理、自定义 Agent 与用户级 Skill 启用/停用；
- Prompt 创建、导入、编辑、复制、导出和删除；
- Project 与个人 Workspace 状态只读展示；
- App Bundle 内置 Go Core，通过私有 stdio JSON-RPC 通信，不监听端口。

Project 的注册、迁移、Link/Copy/Vendor 与个人 Workspace 双向同步属于 Phase 2，首版没有暴露
这些写操作。

## 打开与生成工程

仓库提交了可直接打开的 Xcode workspace：

```bash
open macos/SKM.xcworkspace
```

修改 `Project.swift` 或增删 target 后，使用 Tuist 4.55+ 重新生成：

```bash
tuist generate --path macos --no-open
```

## 构建与测试

```bash
xcodebuild \
  -workspace macos/SKM.xcworkspace \
  -scheme SKM \
  -configuration Debug \
  -derivedDataPath /tmp/skm-macos-derived \
  CODE_SIGNING_ALLOWED=NO \
  build

xcodebuild \
  -workspace macos/SKM.xcworkspace \
  -scheme SKM \
  -configuration Debug \
  -derivedDataPath /tmp/skm-macos-derived \
  CODE_SIGNING_ALLOWED=NO \
  test
```

`Scripts/build-go-core.sh` 始终构建 arm64 与 x86_64 并合并为 Universal 2 Core，
并把 App 的 `MARKETING_VERSION` 注入 `internal/buildinfo.Version`。有签名身份的 Archive 构建
会先以相同身份签名嵌套 Core，再由 Xcode 签名 App。

## 隔离运行

自动化或手动 QA 不应接触真实个人数据。可以给 Debug App 注入以下环境变量：

```text
SKM_HOME=<temporary>/state
SKM_USER_HOME=<temporary>/user
SKM_PROJECT=<temporary>/project
```

App 正式运行时不设置这些变量，Core 会使用与 CLI 相同的 `~/.skm`。

## 发布前仍需外部配置

Developer ID 签名、公证和 stapling 需要发布者的 Apple Developer Team 与钥匙串凭据，仓库不会
保存这些信息。正式发布时还应在干净的 Intel 与 Apple Silicon Mac 上分别验证 DMG。
