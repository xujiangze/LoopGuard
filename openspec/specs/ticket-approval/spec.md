## ADDED Requirements

### Requirement: 工单列表页
系统 SHALL 提供工单列表页（首页），默认展示当前用户作为审批人的工单列表，支持按状态筛选。

#### Scenario: 默认进入显示待审批工单
- **WHEN** 用户登录后进入首页
- **THEN** 系统调用 `GET /api/v1/tickets` 获取当前用户待审批的工单列表并展示

#### Scenario: 按状态筛选
- **WHEN** 用户点击状态筛选标签（全部/待审批/已完成/已驳回/执行失败）
- **THEN** 系统以对应 status 参数调用 API，刷新列表展示筛选结果

#### Scenario: 空列表
- **WHEN** 当前筛选条件下没有工单
- **THEN** 页面显示"暂无工单"的空状态提示

### Requirement: 工单列表展示字段
工单列表每一行 SHALL 展示程序名（project/name 两行显示）、参数预览、提交来源名称、状态标签、提交时间、操作按钮。列表 SHALL 按紧急度排序：失败状态优先，其次待审批，再次进行中，最后已完成和已驳回。每行 SHALL 有左边框颜色标记（红=失败、黄=待审批、绿=完成、灰=其他）。

#### Scenario: 列表行展示
- **WHEN** 工单列表有数据
- **THEN** 每行显示：程序（两行，上 project 下 name）、参数预览（args join 截断）、提交来源名称、状态标签（带颜色）、提交时间、"查看"按钮

#### Scenario: 失败工单视觉突出
- **WHEN** 工单状态为 `exec_failed` 或 `dryrun_failed`
- **THEN** 该行左边框为红色（3px），排在列表最前

#### Scenario: 待审批工单排序
- **WHEN** 工单状态为 `pending_approval`
- **THEN** 该行左边框为黄色，排在失败工单之后

#### Scenario: 默认展示全部工单并显示计数
- **WHEN** 用户进入工单列表页
- **THEN** 前端一次拉取全部工单（不带 status 参数），Tab 标签显示各状态计数（如"待审批(3)"、"失败(2)"），默认选中"全部"

### Requirement: 工单详情页
系统 SHALL 提供工单详情页，展示程序信息、AI 提交参数、dry-run 输出，以及审批/驳回操作。

#### Scenario: 查看工单详情
- **WHEN** 用户从列表点击"查看"进入工单详情页
- **THEN** 页面展示：基本信息（程序名、状态、提交时间）、AI 提交的参数（JSON 格式化展示）、dry-run 输出（保留原始格式，代码块展示）

#### Scenario: 工单不存在
- **WHEN** 用户访问一个不存在的工单 ID
- **THEN** 页面显示"工单不存在"提示，提供返回列表链接

### Requirement: 审批操作
工单详情页 SHALL 在状态为"待审批"时提供"批准执行"和"驳回"按钮。

#### Scenario: 批准执行
- **WHEN** 用户在待审批工单详情页点击"批准执行"按钮
- **THEN** 系统调用 `POST /api/v1/tickets/:id/approve`，成功后跳回工单列表页

#### Scenario: 审批失败
- **WHEN** 审批请求返回错误（如无权限）
- **THEN** 页面显示错误提示，停留在详情页

### Requirement: 驳回操作
工单详情页 SHALL 在用户点击驳回时弹出 Dialog，要求输入驳回原因，确认后调用驳回接口。

#### Scenario: 驳回工单
- **WHEN** 用户点击"驳回"按钮
- **THEN** 弹出 Dialog 要求输入驳回原因（文本输入框），用户填写原因并确认后，调用 `POST /api/v1/tickets/:id/reject`，成功后跳回列表

#### Scenario: 驳回原因为空
- **WHEN** 用户在驳回 Dialog 中不输入原因直接确认
- **THEN** 允许提交（原因非必填），调用驳回接口

### Requirement: 非待审批状态的详情页
工单详情页 SHALL 根据工单状态展示不同内容：已完成显示执行结果，已驳回显示驳回原因。

#### Scenario: 已完成工单
- **WHEN** 用户查看状态为"已完成"的工单
- **THEN** 展示执行结果（退出码、耗时、stdout/stderr），隐藏审批/驳回按钮

#### Scenario: 已驳回工单
- **WHEN** 用户查看状态为"已驳回"的工单
- **THEN** 展示驳回原因，隐藏审批/驳回按钮

#### Scenario: Dry-run 失败工单
- **WHEN** 用户查看状态为"dryrun 失败"的工单
- **THEN** 展示 dry-run 输出和失败原因，隐藏审批/驳回按钮

### Requirement: 审批人查看工单列表
工单列表 SHALL 展示每条工单的提交来源名称。名称由后端在 `submitted_by_name` 字段中直接返回，前端 SHALL NOT 请求 `/api-keys` 接口。当 API Key 已被删除或找不到映射时，SHALL 显示 `Key #id` 作为 fallback。

#### Scenario: 列表显示 API Key 名称
- **WHEN** 审批人打开工单列表页，存在由名称为 "my-agent" 的 API Key 提交的工单
- **THEN** 列表中该工单行的"提交来源"列直接使用后端返回的 `submitted_by_name` 显示 "my-agent"

#### Scenario: API Key 已删除的 fallback
- **WHEN** 审批人打开工单列表页，某工单的 `submitted_by` 对应的 API Key 已被删除
- **THEN** 列表中该工单行的"提交来源"列显示后端返回的 `Key #3`（fallback 由后端生成）

### Requirement: 审批人查看工单详情
工单详情页的基本信息卡 SHALL 展示"提交来源"行，显示 API Key 名称，映射逻辑与列表页一致。

#### Scenario: 详情页显示 API Key 名称
- **WHEN** 审批人打开工单详情页，该工单由名称为 "deploy-bot" 的 API Key 提交
- **THEN** 基本信息卡中显示 "提交来源: deploy-bot"
