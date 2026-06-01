## MODIFIED Requirements

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

## MODIFIED Requirements

### Requirement: 审批人查看工单列表
工单列表 SHALL 展示每条工单的提交来源名称。名称由后端在 `submitted_by_name` 字段中直接返回，前端 SHALL NOT 请求 `/api-keys` 接口。当 API Key 已被删除或找不到映射时，SHALL 显示 `Key #id` 作为 fallback。

#### Scenario: 列表显示 API Key 名称
- **WHEN** 审批人打开工单列表页，存在由名称为 "my-agent" 的 API Key 提交的工单
- **THEN** 列表中该工单行的"提交来源"列直接使用后端返回的 `submitted_by_name` 显示 "my-agent"

#### Scenario: API Key 已删除的 fallback
- **WHEN** 审批人打开工单列表页，某工单的 `submitted_by` 对应的 API Key 已被删除
- **THEN** 列表中该工单行的"提交来源"列显示后端返回的 `Key #3`（fallback 由后端生成）
