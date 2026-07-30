# skm 发布指南

skm 通过 GitHub Release 提供预编译文件，并由同一个 Tag 自动更新 Homebrew Cask。
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

不能使用默认 `GITHUB_TOKEN` 更新另一个仓库；跨仓库发布需要单独 Token。

## 2. 发布版本

确认 `main` 上的测试通过并且工作区干净，然后创建语义化版本 Tag：

```bash
git tag -a v0.2.0 -m "v0.2.0"
git push origin v0.2.0
```

`.github/workflows/release.yml` 将自动：

1. 运行 Go 测试和安装器测试；
2. 构建 macOS/Linux 的 amd64、arm64 文件；
3. 生成 `checksums.txt` 和 GitHub Release；
4. 更新 `homebrew-tap` 仓库中的 `Casks/skm.rb`。

Release 构建通过链接器把 Tag 版本注入 `skm version`。本地源码构建显示 `dev`；
通过 `go install ...@<version>` 安装时会从 Go 模块构建信息读取版本。

## 3. 发布后验证

Homebrew：

```bash
brew update
brew install --cask zzzzzyijie/tap/skm
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

Snapshot 不会创建 GitHub Release 或更新 Homebrew Tap。

安装器自身不依赖网络的测试：

```bash
sh scripts/install_test.sh
```
