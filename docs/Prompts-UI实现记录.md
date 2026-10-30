# Prompts UI 第一版实现

2026-10-29，按 [设计规划 v2](Prompts-UI设计规划-v2.md) 的方案 A 实施。

本轮按反馈调整为文件夹式侧栏：分类直接放在 Prompts 行下方，点击展开／收起；展开状态独立于当前模块，切换页面后保留。移除独立的“Prompt 分类”区域，统一导航边距；Projects 位于展开内容之后，侧栏整体滚动。收起不清除当前筛选和正文。此轮仅做编译与变更检查，不重跑下方首版回归矩阵。

## 已实现

- **分类与列表**：侧栏提供全部、未分类、标签与数量；中栏只显示当前筛选结果，搜索名称、描述、标签，与分类取交集。支持名称自然升降序、空结果恢复入口、选择校正和保存／导入后定位。
- **阅读详情**：移除正文外层卡片和重复操作，统一标题、元数据、模式切换与正文的左边线。复制正文为主要操作，编辑为次操作，导出、快速查看与删除进入更多菜单；保留复制反馈、读取失败重试和请求过期保护。
- **编辑器**：默认 880 × 580，属性区 248 pt；正文输入与标题采用相同 inset。编辑名称只读，新建支持名称、重名和描述校验；无修改不能保存。标签可搜索、多选、草稿内创建，取消不写入资料库。保留变量定义和并发冲突处理。
- **内容呈现**：共享 Markdown 预览支持标题、段落、列表、引用和围栏代码块；源文与复制使用保存的正文。预览时禁用隐藏编辑器，避免继续修改不可见的内容。
- **细节**：中英文文案、深浅色语义配色、按钮辅助功能名称；修正原生 List 的额外 cell 边距；标签弹窗的结果区域保持固定高度，避免搜索时移动输入框。

分类和搜索状态由 `PromptBrowserState` 持有；读写仍通过 `AppModel` 调用 Go Core。没有变更存储格式或持久化规则。

## 实际规则与范围

2026-10-30 更新：Skills / Prompts 新增或导入不再自动添加 `general`。未设置标签时保持空标签，直接归入未分类；已有条目和 Git 来源保存的标签保留。编辑清空标签的规则保持一致。

本版没有增加收藏、历史界面、正文全文搜索或变量填写。Markdown 预览是常用语法子集，不是完整 CommonMark 渲染器。名称排序、搜索和分类是当前窗口状态，不新增持久化偏好。

## 验证结果

所有测试使用独立临时资料库和偏好域。

| 验证 | 结果 |
| --- | --- |
| `tuist generate --path macos --no-open` | 通过，新源文件已加入工程 |
| `go test ./...` | 通过 |
| `go build ./...` | 通过 |
| macOS Debug 构建 | 通过；最终编译也由下述单元测试完成 |
| `xcodebuild … -only-testing:SKMTests test` | 最新代码 31 项通过，0 失败 |
| `testChineseSkillAgentAndPromptWriteFlow` | 通过，包含 Skill / Agent / Prompt 写入和详情动作回归 |
| `testPromptDraftProtectionAndDetailActions` | 通过，包含草稿保护与精确正文复制检查 |
| `testPromptClassificationAndEditorInBothLanguages` | 尚未通过完整用例；中文创建、分类、未分类、只读名称与未修改禁用保存已走通，后续草稿标签搜索时弹窗失焦 |

UI 自动化多次遇到系统输入法授权和其他应用窗口干扰。未授予输入法权限，已停止前台重试。标签弹窗已修复零高度问题并固定搜索结果区域；完整中英文、深浅色流程仍需在桌面焦点稳定时复测。最后的列表边距和弹窗高度微调已经编译验证，尚未重新截图核对。

已检查实际中文浅色主界面与 880 × 580 编辑弹窗截图。设计规划中的最小窗口、大数据量、放大文字、提高对比度等完整矩阵尚未全部验证，不能视为已验收。

## 主要实现入口

- `macos/SKMApp/Sources/Features/PromptsViews.swift`：列表和行菜单。
- `macos/SKMApp/Sources/Features/PromptFilterSidebar.swift`：分类侧栏。
- `macos/SKMApp/Sources/Features/PromptDetailView.swift`：阅读与详情操作。
- `macos/SKMApp/Sources/Features/PromptEditorSheet.swift`：编辑器。
- `macos/SKMApp/Sources/Features/PromptTagPicker.swift`：草稿标签选择。
- `macos/SKMApp/Sources/Features/PromptDesign.swift`：布局尺寸。
- `macos/SKMApp/Sources/App/AppModel.swift`：选择校正和保存定位。

打开 `macos/SKM.xcworkspace`，选择 `SKM` Scheme 可运行本版。具体隔离运行和构建命令见 [macOS README](../macos/README.md)。
