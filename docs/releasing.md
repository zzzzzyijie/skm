# skm 发布指南

skm 通过 GitHub Release 提供预编译文件，并由同一个 Tag 通过 GitHub Contents API 自动更新 Homebrew Formula。
curl 安装器也从 GitHub Release 下载相同文件并验证 SHA-256。

## 1. 一次性配置 Homebrew Tap

在 GitHub 创建公开仓库：

```text
zzzzzyijie/homebrew-tap
```

仓库默认分支必须是 `main`。创建一个只对该仓库拥有 Contents Read and Write 权限的
fine-grained Personal Access Token，并在 `zzzzzyijie/skm` 仓库中添加 Actions Secret：

```text
HOMEBREW_TAP_GITHUB_TOKEN
```

不能使用默认 `GITHUB_TOKEN` 更新另一个仓库；跨仓库发布需要单独 Token。发布工作流使用该
Token 调用 Contents API，而不依赖跨仓库 Git checkout 或 push。
将仓库改为公开只解决匿名下载，不会赋予 Actions 跨仓库写权限。fine-grained Token
必须满足以下配置：

- Resource owner 是 `zzzzzyijie`；
- Repository access 包含 `homebrew-tap`；
- Repository permissions 的 Contents 是 Read and write；
- `skm` 中的 Secret 名称严格为 `HOMEBREW_TAP_GITHUB_TOKEN`。

如果 Release 产物已成功，但工作流在 Homebrew 步骤失败，更新 Secret 后可在下一个 Tag
验证。也可以根据该 Release 的 `checksums.txt` 手动生成并提交 Formula：

```bash
sh scripts/generate_homebrew_formula.sh \
  v0.2.1 \
  ./checksums.txt \
  ../homebrew-tap/Formula/skm.rb
```

## 2. 发布版本

发布前应先按 [隔离开发与发布流程](development-release-workflow.md) 完成开发二进制冒烟测试、
自动化验证和 GoReleaser Snapshot。确认 `main` 上的测试通过并且工作区干净，然后创建语义化版本 Tag：

```bash
git tag -a v0.2.0 -m "v0.2.0"
git push origin v0.2.0
```

`.github/workflows/release.yml` 将自动：

1. 运行 Go 测试和安装器测试；
2. 构建 macOS/Linux 的 amd64、arm64 文件；
3. 生成 `checksums.txt` 和 GitHub Release；
4. 根据 Release 校验值生成并通过 Contents API 更新 `homebrew-tap` 仓库中的 `Formula/skm.rb`。

Release 构建通过链接器把 Tag 版本注入 `skm version`。本地源码构建显示 `dev`；
通过 `go install ...@<version>` 安装时会从 Go 模块构建信息读取版本。

当前 macOS Release 没有使用 Apple Developer ID 公证。Homebrew Formula 会先验证发布包的
SHA-256，再清除解压文件继承的 quarantine 扩展属性，避免 Gatekeeper 阻止已验证的 CLI。
以后接入 Developer ID 签名和公证后，应移除 Formula 生成器中的 `xattr` 兼容逻辑。

## 3. 发布后验证

Homebrew：

```bash
brew update
brew install zzzzzyijie/tap/skm
skm version
```

curl：

```bash
curl -fsSL https://raw.githubusercontent.com/zzzzzyijie/skm/main/scripts/install.sh | sh
skm version
```

安装指定版本：

```bash
curl -fsSL https://raw.githubusercontent.com/zzzzzyijie/skm/main/scripts/install.sh | \
  sh -s -- --version v0.2.0
```

## 4. 本地检查发布配置

安装 GoReleaser 后执行：

```bash
goreleaser check
goreleaser release --snapshot --clean
```

Snapshot 不会创建 GitHub Release 或更新 Homebrew Tap。Formula 生成器可以单独测试：

```bash
sh scripts/generate_homebrew_formula_test.sh
```

安装器自身不依赖网络的测试：

```bash
sh scripts/install_test.sh
```
