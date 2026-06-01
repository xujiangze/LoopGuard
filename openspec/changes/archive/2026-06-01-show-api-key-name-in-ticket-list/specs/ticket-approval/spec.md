## MODIFIED Requirements

### Requirement: 审批人查看工单列表
工单列表 SHALL 展示每条工单的提交来源（API Key 名称）。页面加载时 SHALL 请求 `/api-keys` 接口建立 `id → name` 映射。当 API Key 已被删除或找不到映射时，SHALL 显示 `Key #id` 作为 fallback。

#### Scenario: 列表显示 API Key 名称
- **WHEN** 审批人打开工单列表页，存在由名称为 "my-agent" 的 API Key 提交的工单
- **THEN** 列表中该工单行的"提交来源"列显示 "my-agent"

#### Scenario: API Key 已删除的 fallback
- **WHEN** 审批人打开工单列表页，某工单的 `submitted_by` 对应的 API Key 已被删除
- **THEN** 列表中该工单行的"提交来源"列显示 "Key #3"（其中 3 是 submitted_by ID）

### Requirement: 审批人查看工单详情
工单详情页的基本信息卡 SHALL 展示"提交来源"行，显示 API Key 名称，映射逻辑与列表页一致。

#### Scenario: 详情页显示 API Key 名称
- **WHEN** 审批人打开工单详情页，该工单由名称为 "deploy-bot" 的 API Key 提交
- **THEN** 基本信息卡中显示 "提交来源: deploy-bot"
