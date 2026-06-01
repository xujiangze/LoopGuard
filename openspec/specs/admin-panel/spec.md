## ADDED Requirements

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

### Requirement: 用户管理页面
系统 SHALL 提供用户管理页面（admin only），展示用户列表并提供创建用户表单。

#### Scenario: 查看用户列表
- **WHEN** admin 用户访问 `/admin/users`
- **THEN** 页面展示用户列表：ID、用户名、角色、创建时间

#### Scenario: 创建用户
- **WHEN** admin 填写创建用户表单（用户名、密码、角色选择）并提交
- **THEN** 系统调用 `POST /api/v1/users`，成功后刷新列表

#### Scenario: 创建用户失败
- **WHEN** 创建请求返回错误（如用户名重复、密码太短）
- **THEN** 页面显示具体错误信息

### Requirement: API Key 管理页面
系统 SHALL 提供 API Key 管理页面（admin only），展示 Key 列表并提供创建功能。创建后的明文 Key 只展示一次。

#### Scenario: 查看 API Key 列表
- **WHEN** admin 用户访问 `/admin/api-keys`
- **THEN** 页面展示 API Key 列表：ID、名称、启用状态、创建时间（不展示哈希）

#### Scenario: 创建 API Key
- **WHEN** admin 输入 Key 名称并提交创建
- **THEN** 系统调用 `POST /api/v1/api-keys`，成功后弹出 Alert Dialog 展示明文 Key

#### Scenario: 明文 Key 一次性展示
- **WHEN** API Key 创建成功后
- **THEN** 弹出醒目的 Alert Dialog，包含：警告文案"请立即复制，此密钥只显示一次"、明文 Key 文本框、复制到剪贴板按钮、"我已保存，关闭"按钮。用户必须点击关闭按钮才能关闭弹窗

#### Scenario: 复制到剪贴板
- **WHEN** 用户在明文展示弹窗中点击"复制到剪贴板"按钮
- **THEN** 明文 Key 复制到系统剪贴板，按钮文案变为"已复制"

### Requirement: 管理页面的权限控制
所有 `/admin/*` 页面 SHALL 仅对 role 为 "admin" 的用户可见和可访问。

#### Scenario: admin 访问管理页面
- **WHEN** admin 用户访问 `/admin/programs`、`/admin/users` 或 `/admin/api-keys`
- **THEN** 正常展示页面内容

#### Scenario: 侧边栏权限控制
- **WHEN** role 为 "user" 的用户登录
- **THEN** 侧边栏不显示"管理"分组（程序管理、用户管理、API Key 管理入口全部隐藏）
