## MODIFIED Requirements

### Requirement: 程序管理页面
系统 SHALL 提供程序管理页面（admin only），展示已注册程序列表，并提供注册新程序的表单。表单和列表 SHALL 包含 interpreter 字段。

#### Scenario: 查看程序列表
- **WHEN** admin 用户访问 `/admin/programs`
- **THEN** 页面展示所有已注册程序列表：项目/名称、解释器+二进制路径、审批人、启用状态、操作按钮。当 interpreter 非空时，路径展示为 `interpreter binary_path` 格式

#### Scenario: 注册新程序
- **WHEN** admin 点击"注册新程序"并填写表单（项目名、程序名、解释器（可选）、二进制路径、审批人、超时、参数白名单）
- **THEN** 系统调用 `POST /api/v1/programs`，请求体包含 `interpreter` 字段，成功后刷新列表

#### Scenario: 注册失败
- **WHEN** 注册请求返回错误（如程序未识别 --only-print）
- **THEN** 页面显示具体错误信息

#### Scenario: 启用/禁用程序
- **WHEN** admin 在程序列表中修改某个程序的启用状态
- **THEN** 系统调用 `PUT /api/v1/programs/:id` 更新 enabled 字段

#### Scenario: 编辑程序的 interpreter
- **WHEN** admin 在编辑弹窗中修改 interpreter 字段并提交
- **THEN** 系统调用 `PUT /api/v1/programs/:id` 更新 interpreter 字段
