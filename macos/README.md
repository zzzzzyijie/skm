# SKM macOS 原生 App

当前原生首版为 `0.5.1`，实现技术设计中的 Core 契约与原生 MVP 主流程：

- 支持跟随 macOS 系统语言显示简体中文或英文；

- SwiftUI 三栏导航与 macOS 原生菜单、搜索、文件面板、剪贴板和 Settings scene；
- Skill 本地目录/ZIP/Git/安装命令导入、详情、原文编辑、标签、删除；
- Agent 检测、管理、自定义 Agent 与用户级 Skill 启用/停用；
- Prompt 创建、导入、编辑、复制、导出和删除；
- 首次启动识别现有资料库、Core 阻断错误页、脱敏诊断复制；
- 主操作菜单快捷键、VoiceOver 行摘要与隔离 XCUITest；
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
  test
```

UI Test 需要 Xcode 的本地 ad-hoc 签名，因此 `test` 不要传 `CODE_SIGNING_ALLOWED=NO`。所有 UI Test
都会设置临时的 `SKM_HOME`、`SKM_USER_HOME` 与 `SKM_PROJECT`，不会读取或修改真实个人数据。

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

## 构建发布候选

不使用 Apple 凭据也能生成本地预览包，用于检查 Universal 2、App/Core 版本和 DMG 内容：

```bash
sh macos/Scripts/package-release.sh \
  --version 0.5.1 \
  --build 1 \
  --output dist/macos \
  --preview
```

`--preview` 使用本地 ad-hoc 签名并验证 App 完整性，但产物不能公开分发。正式打包需要设置：

```text
SKM_SIGNING_IDENTITY=Developer ID Application: <Name> (<TEAM_ID>)
SKM_NOTARY_KEY_PATH=/absolute/path/to/AuthKey_<KEY_ID>.p8
SKM_NOTARY_KEY_ID=<KEY_ID>
SKM_NOTARY_ISSUER_ID=<ISSUER_UUID>
```

然后去掉 `--preview`。脚本会按顺序构建 Universal 2 App、签名 Core 与 App、用 `notarytool`
公证、staple、生成 ZIP/DMG，并执行 Gatekeeper 与校验和检查。

## 发布前仍需外部配置

Developer ID 签名、公证和 stapling 需要发布者的 Apple Developer Team 与钥匙串凭据，仓库不会
保存这些信息。Tag 发布工作流要求在 GitHub Actions 中配置以下 Secrets：

```text
MACOS_DEVELOPER_ID_P12_BASE64
MACOS_DEVELOPER_ID_P12_PASSWORD
MACOS_NOTARY_PRIVATE_KEY_BASE64
MACOS_NOTARY_KEY_ID
MACOS_NOTARY_ISSUER_ID
```

前两个来自包含私钥的 Developer ID Application `.p12`；后三个来自 App Store Connect Team API
Key。正式发布时还应在一台没有 Go/Homebrew/SKM CLI 的 Intel Mac 与 Apple Silicon Mac 上分别
验证 DMG。Release 工作流已经使用干净的 `macos-26` 与 `macos-26-intel` runner 自动执行两种
架构的签名、stapling、Gatekeeper、内置 Core 握手和 App 启动检查；发布后仍建议进行一次人工安装。
