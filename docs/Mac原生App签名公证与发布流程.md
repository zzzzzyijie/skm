# SKM macOS 原生 App 签名、公证与发布流程

> 最后验证日期：2026-08-27
>
> 验证结论：本地 Developer ID 签名、Apple 公证、stapling、Gatekeeper 与校验和闭环已通过。
>
> 验证产物：`SKM 0.6.0 (Build 3)`，仅用于发布链路验收，不代表已经创建 `v0.6.0` Tag 或 GitHub Release。

本文记录 SKM 原生 macOS App 在 Mac App Store 之外分发时的完整流程。最终用户安装 DMG 或 ZIP
中的 `SKM.app`，不需要安装 Go、Homebrew 或 SKM CLI；Go 只在开发和打包时用于生成内置
Universal 2 `skm-core`。

相关文件：

- 发布脚本：[`macos/Scripts/package-release.sh`](../macos/Scripts/package-release.sh)
- 本机配置模板：[`macos/.release.env.example`](../macos/.release.env.example)
- macOS 工程说明：[`macos/README.md`](../macos/README.md)
- macOS 开发进度：[`docs/Mac原生版开发进度.md`](Mac原生版开发进度.md)
- GitHub Release 工作流：[`.github/workflows/release.yml`](../.github/workflows/release.yml)

## 1. Apple 侧一次性准备

### 1.1 Developer ID Application

在 Apple Developer 的 Certificates, Identifiers & Profiles 中创建：

```text
Certificate Type: Developer ID Application
Profile Type: G2 Sub-CA (Xcode 11.4.1 or later)
```

SKM 使用 DMG/ZIP 分发 App，因此需要 `Developer ID Application`。只有以后改为 `.pkg` 安装包时，
才需要额外申请 `Developer ID Installer`。

`G2 Sub-CA` 是现代 Xcode 的正常选择。`Previous Sub-CA` 仅用于必须在 Xcode 11.4 或更早版本中
签名的遗留构建环境，不用于控制 App 的最低 macOS 版本。

下载 `.cer` 后双击导入钥匙串。证书必须同时拥有对应私钥，以下命令应能找到有效身份：

```bash
security find-identity -v -p codesigning
```

签名身份是完整证书名称，不是单独的 Team ID：

```text
Developer ID Application: <Organization or Name> (<TEAM_ID>)
```

### 1.2 App Store Connect Team API Key

进入 App Store Connect：

```text
Users and Access → Integrations → App Store Connect API → Team Keys
```

创建 Team API Key，并记录：

- 下载一次后必须妥善保存的 `AuthKey_<KEY_ID>.p8`；
- Team Keys 列表中的 Key ID；
- API Keys 页面顶部、UUID 格式的 Issuer ID。

必须使用能够调用 `notarytool` 的 Team Key，不能使用 Individual API Key。Team Key 应与
Developer ID Application 所属 Apple Developer 团队一致。

## 2. 本机凭据配置

复制模板：

```bash
cp macos/.release.env.example macos/.release.env.local
chmod 600 macos/.release.env.local
```

填写 `macos/.release.env.local`：

```bash
export SKM_SIGNING_IDENTITY='Developer ID Application: <Name> (<TEAM_ID>)'
export SKM_NOTARY_KEY_PATH='/absolute/path/to/AuthKey_<KEY_ID>.p8'
export SKM_NOTARY_KEY_ID='<KEY_ID>'
export SKM_NOTARY_ISSUER_ID='<ISSUER_UUID>'
```

收紧私钥权限：

```bash
chmod 600 "$SKM_NOTARY_KEY_PATH"
```

安全要求：

- `macos/.release.env.local`、`*.p8` 和 `*.p12` 已被 `.gitignore` 忽略；
- `.p8` 最好保存在仓库之外，本机配置只记录绝对路径；
- 不把证书密码、私钥、Key ID、Issuer ID 或 Base64 内容写进文档、Issue、PR 和构建日志；
- 不使用 `git add -f` 强制添加任何签名凭据；
- 怀疑私钥泄露时，立即在 Apple 后台撤销并重新创建。

确认 Git 不追踪本机配置或私钥：

```bash
git check-ignore -v macos/.release.env.local
git ls-files '*.p8' '*.p12' macos/.release.env.local
```

第二条命令应没有输出。

## 3. 发布前认证检查

发布脚本会自动加载 `macos/.release.env.local`。也可以先做一次不会上传 App 的认证检查：

```bash
. macos/.release.env.local

test -f "$SKM_NOTARY_KEY_PATH"
openssl pkey -in "$SKM_NOTARY_KEY_PATH" -noout

xcrun notarytool history \
  --key "$SKM_NOTARY_KEY_PATH" \
  --key-id "$SKM_NOTARY_KEY_ID" \
  --issuer "$SKM_NOTARY_ISSUER_ID"
```

`notarytool history` 能正常返回，说明 `.p8`、Key ID 和 Issuer ID 可以完成 Apple 公证认证。
它只读取历史记录，不提交新产物。

## 4. 构建模式

### 4.1 无凭据本地预览

```bash
sh macos/Scripts/package-release.sh \
  --version 0.5.2 \
  --build 1 \
  --output dist/macos-preview \
  --preview
```

预览模式使用 ad-hoc 签名，不提交 Apple 公证，仅用于本机检查，不可公开分发。

### 4.2 正式签名与公证

```bash
sh macos/Scripts/package-release.sh \
  --version 0.5.2 \
  --build 1 \
  --output dist/macos
```

正式发布时，`--version` 应与将要创建的 Git Tag 一致，`--build` 必须是正整数并随发布递增。

脚本按以下顺序执行：

```text
构建 arm64 与 x86_64 Go Core
→ 合并 Universal 2 Core
→ 构建 Universal 2 SKM.app
→ 检查 App/Core/参数版本一致
→ Developer ID 签名内置 Core
→ Developer ID 签名 SKM.app
→ 上传 App 压缩包进行公证
→ 将公证票据 staple 到 SKM.app
→ 生成最终 ZIP 与 DMG
→ Developer ID 签名 DMG
→ 上传 DMG 进行公证
→ 将公证票据 staple 到 DMG
→ Gatekeeper 检查 App 与 DMG
→ 生成 ZIP/DMG SHA-256 校验文件
```

签名顺序不能调整：嵌套 Core 必须先于外层 App；DMG 必须在提交公证之前签名；已经 staple 的
文件不能再次修改或重签，否则公证票据会失效。

## 5. 成功标准

App 与 DMG 的公证提交都应返回：

```text
status: Accepted
```

Stapler 应返回：

```text
The staple and validate action worked!
```

Gatekeeper 应同时接受 App 和 DMG：

```text
accepted
source=Notarized Developer ID
```

校验和验证：

```bash
cd dist/macos
shasum -a 256 -c SKM-<version>-checksums.txt
```

ZIP 和 DMG 都必须显示 `OK`。

## 6. 2026-08-27 实际验收记录

本次使用真实 Developer ID Application 和 App Store Connect Team API Key，执行：

```bash
sh macos/Scripts/package-release.sh \
  --version 0.6.0 \
  --build 3 \
  --output dist/macos-notarized
```

结果：

| 检查项 | 结果 |
| --- | --- |
| 本机配置、证书和 API Key 认证 | 通过 |
| App/Core Universal 2 | `arm64 + x86_64` |
| Core 与 App Developer ID 签名 | 通过 |
| App 公证 | `Accepted` |
| App stapling | 通过 |
| App Gatekeeper | `accepted / Notarized Developer ID` |
| DMG Developer ID 签名 | 通过 |
| DMG 公证 | `Accepted` |
| DMG stapling | 通过 |
| DMG Gatekeeper | `accepted / Notarized Developer ID` |
| ZIP、DMG SHA-256 | 通过 |
| `go test ./...`、`go build ./...` | 通过 |

最终成功提交记录：

```text
App submission: bf3063a2-6e61-4720-87ca-ceebcfebd52a
DMG submission: cbb861c0-e619-4777-9dcf-6a43cd7e2e51
```

这两个 Submission ID 仅用于在 Apple 公证历史中审计本次验收，不是凭据。

### 本次发现并修复的问题

首次运行时，App 与 DMG 公证、stapling 均成功，但 DMG 的 Gatekeeper 检查返回：

```text
rejected
source=no usable signature
```

根因是 DMG 容器在提交公证前没有执行 Developer ID 签名。Apple 接受公证提交并不代表 DMG
自动获得代码签名。修复后，发布脚本在 DMG 生成后、公证前执行：

```bash
codesign --force --timestamp --sign "$SKM_SIGNING_IDENTITY" "$DMG_PATH"
codesign --verify --verbose=2 "$DMG_PATH"
```

重新提交后，DMG 公证、stapling 和 Gatekeeper 全部通过。

## 7. GitHub Actions 正式发布

本机配置不会被 CI 使用。GitHub 仓库需要单独配置 Actions Secrets：

```text
MACOS_DEVELOPER_ID_P12_BASE64
MACOS_DEVELOPER_ID_P12_PASSWORD
MACOS_NOTARY_PRIVATE_KEY_BASE64
MACOS_NOTARY_KEY_ID
MACOS_NOTARY_ISSUER_ID
```

Tag 发布工作流会把 `.p12` 导入临时钥匙串，把 `.p8` 写入 runner 临时目录，调用同一个
`package-release.sh`，然后在 Apple Silicon 与 Intel runner 上验证产物。凭据必须通过 GitHub
Actions Secrets 提供，不能提交到仓库。

## 8. 正式发布前剩余人工验收

本地发布技术闭环已完成，正式创建 Tag 和 GitHub Release 前仍应：

- 确认正式版本号、Build 号和 Bundle ID；
- 在没有 Go、Homebrew 和 SKM CLI 的干净 Apple Silicon Mac 上安装并启动；
- 在没有 Go、Homebrew 和 SKM CLI 的干净 Intel Mac 上安装并启动；
- 验证首次启动、Core 握手、Skills、Agents 和 Prompts 主要读写流程；
- 验证中英文、VoiceOver、键盘、浅色、深色和窄窗口；
- 配置并验证 GitHub Actions Secrets；
- 创建正式 Tag 后检查 GitHub Release、DMG、ZIP 与校验和。

## 9. Apple 官方参考

- [创建 Developer ID 证书](https://developer.apple.com/help/account/certificates/create-developer-id-certificates/)
- [Developer ID 中间证书更新](https://developer.apple.com/support/developer-id-intermediate-certificate/)
- [App Store Connect API 与 Team Keys](https://developer.apple.com/help/app-store-connect/get-started/app-store-connect-api/)
- [自定义 macOS 公证流程](https://developer.apple.com/documentation/security/customizing-the-notarization-workflow)
- [分发前公证 macOS 软件](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution)
