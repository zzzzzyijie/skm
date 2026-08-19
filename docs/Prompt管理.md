# Prompt 管理

## 第一版边界

Prompt 是与 Skill 平级的独立资产：Skill 描述 Agent 应具备的能力，Prompt 描述用户准备
发送的内容。Prompt 不参与 Skill Activation，不会部署到 Agent Skill 目录，第一版也不调用
任何模型 API。

第一版支持：

- 本地 `PROMPT.md` 创建、导入、查看、更新、导出和删除。
- 标签、搜索和不可变内容快照。
- `text`、`multiline`、`number`、`boolean`、`select`、`secret` 变量。
- CLI 严格渲染，以及 Web 变量表单、预览和复制。
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
`{{variable}}` 都必须在 frontmatter 中声明。`secret` 变量不能设置默认值，Web 和 CLI
不会把本次变量输入保存回 Catalog。

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

`prompt render` 有缺失的必填变量时会失败。Web 预览会保留未替换占位符，并列出缺失变量，
所有必填值完整后才能复制最终结果。

## 后续能力

- Prompt Git Source、更新和来源只读保护。
- 历史版本、差异查看和回滚。
- 项目级 Prompt 和 Prompt 组合。
- 独立的 Agent 投递适配层。
