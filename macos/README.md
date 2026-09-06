# SKM macOS 原生 App

当前已完成项、待完成项和正式分发所需信息见
[macOS 原生版开发进度](../docs/Mac原生版开发进度.md)。
Developer ID、Apple 公证、stapling 与 Gatekeeper 的完整实操见
[macOS 原生 App 签名、公证与发布流程](../docs/Mac原生App签名公证与发布流程.md)。

当前工程版本为 `0.5.3`，Phase 1 至 Phase 3 均已完成：

- 支持跟随 macOS 系统语言显示简体中文或英文；
- SwiftUI 三栏主界面聚焦 Skills、Prompts、Projects，侧栏底部和 `⌘,` 可打开独立 Settings 窗口；
- 技能来源、Git 同步、Agent 管理、诊断与更新集中在设置分类中，主工具栏保留统一同步快捷菜单；
- Skill 本地目录/ZIP/Git/安装命令导入、详情、原文编辑、标签、删除；
- Agent 检测、管理、自定义 Agent 与用户级 Skill 启用/停用；
- Prompt 创建、导入、编辑、复制、导出和删除；
- 首次启动识别现有资料库、Core 阻断错误页、Doctor、脱敏诊断导出与更新检查；
- 主操作菜单快捷键、VoiceOver 行摘要与中英文隔离 XCUITest；
- Project 登记、扫描、Link/Copy 部署、预览、冲突阻断、迁移、解绑与注销；
- Source 添加、更新、移除与统一同步；
- 个人 Git 同步配置、双向预览与逐项冲突解决；Core 内部仍沿用 Workspace 契约；
- App Bundle 内置 Go Core，通过私有 stdio JSON-RPC 通信，不监听端口。
- Project Require/Vendor/Apply、Prompt 变量表单与安全渲染、历史差异/回滚、Quick Look；
- Sparkle 2 签名 appcast 自动更新；未注入发布公钥的开发构建会回退到 GitHub 版本检查。

## 打开与生成工程

主界面采用系统三栏布局与侧栏材质，导航显示各资料库数量；Skill / Prompt 详情使用独立阅读区域，Agent 激活按钮显示品牌图标与启用状态。配色跟随系统浅色与深色外观。

Skills、Prompts 和 Projects 均支持 `⌘F` 聚焦搜索。搜索框内按 `Esc` 清空搜索，再按一次移出焦点；项目可按名称或路径查找。同步执行期间会禁用侧栏同步按钮，避免重复触发。

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
  --version 0.5.3 \
  --build 3 \
  --output dist/macos \
  --preview
```

`--preview` 使用本地 ad-hoc 签名并验证 App 完整性，但产物不能公开分发。正式打包需要设置：

```text
SKM_SIGNING_IDENTITY=Developer ID Application: <Name> (<TEAM_ID>)
SKM_NOTARY_KEY_PATH=/absolute/path/to/AuthKey_<KEY_ID>.p8
SKM_NOTARY_KEY_ID=<KEY_ID>
SKM_NOTARY_ISSUER_ID=<ISSUER_UUID>
SKM_SPARKLE_PUBLIC_KEY=<SPARKLE_ED25519_PUBLIC_KEY>
SKM_SPARKLE_PRIVATE_KEY_PATH=/absolute/path/to/skm.sparkle-key
```

也可以复制本机配置模板，发布脚本会自动加载该文件：

```bash
cp macos/.release.env.example macos/.release.env.local
```

`macos/.release.env.local` 已加入 `.gitignore`，可以保存本机证书名称和 `.p8` 路径，不能提交到
Git。外部已经设置的同名环境变量优先于模板中的本机值。

Sparkle 使用独立的 EdDSA 密钥，不是 Developer ID 或公证 `.p8`。首次生成工程并完成一次构建后，
在 Xcode DerivedData 的 `SourcePackages/artifacts/sparkle/Sparkle/bin/` 中运行：

```bash
generate_keys --account com.zzzzzyijie.skm
generate_keys --account com.zzzzzyijie.skm -p
generate_keys --account com.zzzzzyijie.skm -x /安全位置/skm.sparkle-key
```

第一条创建钥匙串密钥，第二条输出可嵌入 App 的公钥，第三条导出用于 CI appcast 签名的私钥。
私钥等同密码，只能放在本机安全位置或 `MACOS_SPARKLE_PRIVATE_KEY_BASE64` Secret；不能提交到 Git。

然后去掉 `--preview`。脚本会按顺序构建 Universal 2 App、签名 Core、App 与 DMG、用 `notarytool`
公证、staple、生成 ZIP/DMG 和 EdDSA 签名 `appcast.xml`，并执行 Gatekeeper 与校验和检查。

## 发布前仍需外部配置

Developer ID 签名、公证和 stapling 需要发布者的 Apple Developer Team 与钥匙串凭据，仓库不会
保存这些信息。Tag 发布工作流要求在 GitHub Actions 中配置以下 Secrets：

```text
MACOS_DEVELOPER_ID_P12_BASE64
MACOS_DEVELOPER_ID_P12_PASSWORD
MACOS_NOTARY_PRIVATE_KEY_BASE64
MACOS_NOTARY_KEY_ID
MACOS_NOTARY_ISSUER_ID
MACOS_SPARKLE_PUBLIC_KEY
MACOS_SPARKLE_PRIVATE_KEY_BASE64
```

前两个来自包含私钥的 Developer ID Application `.p12`；后三个来自 App Store Connect Team API
Key。正式发布时还应在一台没有 Go/Homebrew/SKM CLI 的 Intel Mac 与 Apple Silicon Mac 上分别
验证 DMG。Release 工作流已经使用干净的 `macos-26` 与 `macos-26-intel` runner 自动执行两种
架构的签名、stapling、Gatekeeper、内置 Core 握手和 App 启动检查；发布后仍建议进行一次人工安装。
