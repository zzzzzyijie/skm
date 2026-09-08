# macOS UI 与交互打磨

本次范围为 `macos/SKMApp` 原生 App。Web 界面保持现状。

采用系统三栏导航、系统强调色和浅／深色自适应背景。通过共用组件统一阅读宽度、标题图标、分组卡片、编辑器边界和弹窗反馈；没有引入第三方 UI 依赖，最低系统版本仍为 macOS 14。

## 覆盖范围

| 界面 | 调整 |
| --- | --- |
| 主窗口、侧栏、搜索 | 资料数量、统一列表页脚、搜索焦点、扩大底栏按钮响应区域 |
| Skills | 标题与阅读层次、Agent 卡片按压／悬停反馈、设置入口、代码块复制 |
| Prompts | 独立复制工具区、变量渲染入口、阅读层次、加载中操作禁用 |
| Projects | 文件夹摘要行、概览卡片、纵向详情标题、按项目隔离部署预览、移除部署确认 |
| 通用、权限、Agents、来源、Git 同步、更新 | 统一标题与卡片、窄窗口布局、任务状态、面向用户的说明 |
| Skill／Source 导入 | 图标标题、步骤、主操作快捷键、扫描／提交防重复、就地错误 |
| Skill／Prompt 编辑 | 编辑器表面、⌘S、草稿丢弃确认、并发冲突保留、变量区域滚动 |
| 项目登记、迁移、导入、自定义 Agent | 统一弹窗结构、表单校验、Return／Esc、忙碌状态 |
| 标签管理、新增、重命名 | 共用搜索、标题、快捷键、删除按钮辅助功能标签 |
| 变量渲染、历史版本 | 预览卡片、必填提示、结果失效处理、取消旧版本读取 |
| 欢迎、空状态、错误、状态提示 | 系统组件、引导说明、加载指示与成功图标区分 |

## 交互规则

- `⌘1/2/3` 切换集合，`⌘F` 搜索，`⌘N` 新建，`⌘O` 导入。
- `⌘S` 保存编辑，`⌘Y` 快速查看，`⌘,` 打开设置。
- 编辑草稿有变动时，取消先确认；无变动直接关闭。
- 搜索框内 Esc 先清空搜索，再移出焦点；普通弹窗 Esc 取消。
- 提交期间禁用重复操作，在当前弹窗显示任务或错误。
- 选择卡片按压缩放为 0.96，悬停／按压过渡 150ms；减少动态效果时取消动画与缩放。选中状态始终由图标、文字或边界同时表达。

## 修复记录

| 严重度 | 位置 | 之前 | 之后 | 原则与影响 |
| --- | --- | --- | --- | --- |
| HIGH | `Features/SkillsViews.swift`、`Features/PromptsViews.swift` | 关闭编辑器会丢弃草稿 | 有修改时确认，支持继续编辑 | 保留用户输入 |
| HIGH | `App/SKMApp.swift` | 全局空格键绑定预览 | 使用 ⌘Y | 不干扰文本输入 |
| HIGH | `Features/Phase3Views.swift` | 改变量后仍可复制旧渲染结果 | 清除旧结果并禁用复制，校验异步结果对应输入 | 操作状态与内容一致 |
| MEDIUM | `Features/SkillsViews.swift`、`Features/PromptsViews.swift`、`Features/Phase3Views.swift`、`App/AppModel.swift` | 旧异步读取可能覆盖当前详情 | 检查取消状态及当前选择 | 快速切换结果一致 |
| MEDIUM | `Features/PromptsViews.swift` | 复制按钮覆盖正文 | 独立正文工具区 | 内容和操作分区 |
| MEDIUM | `App/RootView.swift`、`App/SheetChrome.swift` | 错误与任务反馈主要位于主窗口 | 设置与弹窗也可见反馈 | 反馈接近操作 |
| MEDIUM | `App/SurfaceStyle.swift`、各 Features | 表面、圆角、阅读宽度不一致 | 共用样式与尺寸 | 视觉结构一致 |

## 验证方法

`SKMUITests` 使用临时资料库、用户目录、项目目录和独立偏好域。新增用例覆盖草稿保护、⌘S、变量预览及中英文／浅深色截图，截图作为 XCTest 附件保留。

```sh
xcodebuild -workspace macos/SKM.xcworkspace -scheme SKM \
  -derivedDataPath /tmp/skm-ui-polish-derived \
  -resultBundlePath /tmp/skm-ui-review.xcresult test
xcrun xcresulttool export attachments --path /tmp/skm-ui-review.xcresult \
  --output-path /tmp/skm-ui-review-screenshots
```

UI 测试需要独占前台键鼠；其他应用抢占焦点会导致点击／输入失败。不要把此类失败记为交互验证通过。

Not verified：macOS 14 真机外观、VoiceOver 实际朗读、系统增强对比度／减少动态效果的人工验收，以及所有网络失败与部署冲突组合。代码层面已检查控件的可用、禁用、加载、空与错误状态；新增 UI 测试不能替代这些人工检查。
