## ADDED Requirements

### Requirement: 工单列表返回富字段
`GET /api/v1/tickets` 列表接口 SHALL 返回 `ticketListItem` 结构，包含 `program_project`、`program_name`、`submitted_by_name` 富字段，前端无需额外请求即可展示程序名称和提交来源。

#### Scenario: 普通用户获取列表
- **WHEN** 普通用户（非 admin）调用 `GET /api/v1/tickets`
- **THEN** 响应中每个工单包含 `program_project`（如 "infra"）、`program_name`（如 "k8s-restart"）、`submitted_by_name`（如 "生产部署Key"）

#### Scenario: 程序已被删除
- **WHEN** 工单关联的 program_id 对应的程序已被删除
- **THEN** `program_project` 和 `program_name` SHALL 为空字符串

#### Scenario: API Key 已被删除
- **WHEN** 工单的 submitted_by 对应的 API Key 已被删除
- **THEN** `submitted_by_name` SHALL 显示 `Key #<id>` 作为 fallback

### Requirement: 列表响应不含大文本字段
`GET /api/v1/tickets` 列表响应 SHALL NOT 包含 `dryrun_output` 和 `exec_output` 字段。这些字段由 `GET /api/v1/tickets/:id` 详情接口提供。

#### Scenario: 列表响应无 output 字段
- **WHEN** 调用 `GET /api/v1/tickets`
- **THEN** 响应 JSON 中不包含 `dryrun_output` 和 `exec_output` 键

### Requirement: store 批量查询方法
Store 层 SHALL 提供 `GetProgramsByIDs(ids []uint64)` 和 `GetAPIKeysByIDs(ids []uint64)` 方法，返回 `map[uint64]model`，用于列表接口批量关联查询。

#### Scenario: 批量查询程序
- **WHEN** 调用 `GetProgramsByIDs([]uint64{1, 3, 5})`
- **THEN** 返回 map，key 为程序 ID，value 为对应的 Program 模型

#### Scenario: 部分ID不存在
- **WHEN** 调用 `GetProgramsByIDs([]uint64{1, 999})` 其中 999 不存在
- **THEN** 返回 map 只包含 ID=1 的条目，不报错
