# Prompt 管理

## 第一版边界

Prompt 是与 Skill 平级的独立资产：Skill 描述 Agent 应具备的能力，Prompt 描述用户准备
发送的内容。Prompt 不参与 Skill Activation，不会部署到 Agent Skill 目录，第一版也不调用
任何模型 API。

第一版支持：

- 本地 `PROMPT.md` 创建、导入、查看、更新、导出和删除。
- 标签、搜索和不可变内容快照。
- `text`、`multiline`、`number`、`boolean`、`select`、`secret` 变量。
- CLI 严格渲染，以及 Web 结构化创建、编辑和正文复制。
- 基于 `baseHash` 的编辑冲突保护。

## 文件格式

```markdown
---
name: code-review
description: 对指定代码进行结构化审查
tags: [development, review]
variables:
  - name: language
    label: 编程语言
    type: select
    required: true
    options: [Go, Swift]
  - name: code
    label: 待审查代码
    type: multiline
    required: true
---
你是一名资深 {{language}} 工程师。

请审查下面的代码：

{{code}}
```

变量名只能使用小写字母、数字、连字符和下划线，并以小写字母开头。正文引用的每一个
`{{variable}}` 都必须在 frontmatter 中声明。`secret` 变量不能设置默认值，CLI
不会把本次变量输入保存回 Catalog。Web 第一版保留导入文件中的变量定义，但卡片主操作会直接
复制 Prompt 正文，其中的占位符保持原样。

## 本地数据

```text
~/.skm/
├── prompt-catalog.yaml
└── prompt-objects/
    └── <hash>/<name>/PROMPT.md
```

Catalog 保存索引和当前 Hash，正文存放在不可变快照中。更新时先创建新快照，再原子更新
Catalog；没有其他引用的旧快照会被清理。

## CLI

```bash
skm prompt create summary --description "Summarize content" \
  --variable content:multiline \
  --body "Summarize: {{content}}"
skm prompt validate ./PROMPT.md
skm prompt add ./PROMPT.md
skm prompt list --tag review
skm prompt show local/code-review
skm prompt update local/code-review ./PROMPT.md --base-hash <hash>
skm prompt render local/code-review --var language=Go --var-file code=./main.go
skm prompt export local/code-review --output ./PROMPT.md
skm prompt remove local/code-review
```

`prompt render` 有缺失的必填变量时会失败。Web 新建弹窗只需填写名称、描述、标签和正文；
可以在 Prompt 编辑器中直接创建并选择标签，未输入标签名时创建按钮保持禁用。Prompt 与
Skill 各自维护独立的标签注册表、用量统计和增删改操作，同名标签也互不影响。新建 Prompt
没有显式选择标签时使用 Prompt 自己的默认 `general` 标签，不会读取 Skill 的默认标签。
旧版共享标签配置会按现有 Skill 与 Prompt 的实际引用自动拆分并持久化；仅被 Prompt 使用的
标签不会继续出现在 Skill 标签管理中。
服务端负责生成 `PROMPT.md` frontmatter；“复制 Prompt”会把正文直接写入设备剪贴板。

标签支持 Unicode 字母和数字（例如 `简小知`、`review-2`），允许在中间使用连字符。
系统会规范化大小写和兼容字符并去重；标签长度为 1～32 个 Unicode 字符，不能包含空格、
下划线、斜杠、emoji、控制字符或首尾连字符。

## 后续能力

- Prompt Git Source、更新和来源只读保护。
- 历史版本、差异查看和回滚。
- 项目级 Prompt 和 Prompt 组合。
- 独立的 Agent 投递适配层。
