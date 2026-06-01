## ADDED Requirements

### Requirement: 登录页面
系统 SHALL 提供登录页面，包含用户名和密码输入框，提交后调用 `POST /api/v1/auth/login` 获取 JWT。

#### Scenario: 登录成功
- **WHEN** 用户输入正确的用户名和密码并点击登录
- **THEN** 系统将 JWT token、角色、用户 ID 存入 localStorage，跳转到首页（工单列表）

#### Scenario: 登录失败
- **WHEN** 用户输入错误的用户名或密码
- **THEN** 页面显示"用户名或密码错误"提示，停留在登录页

#### Scenario: 网络异常
- **WHEN** 登录请求因网络问题失败
- **THEN** 页面显示"网络错误，请稍后重试"提示

### Requirement: JWT 持久化与自动携带
系统 SHALL 将 JWT token 存储在 localStorage 中，并在所有需要认证的 API 请求中自动携带 `Authorization: Bearer <token>` header。

#### Scenario: 已登录用户访问页面
- **WHEN** 用户已登录（localStorage 中有有效 token）并访问任意认证页面
- **THEN** 系统自动携带 token 发起请求，正常展示页面内容

#### Scenario: Token 过期
- **WHEN** 已登录用户的 token 过期，API 返回 401
- **THEN** 系统清除 localStorage 中的认证信息，跳转到登录页

### Requirement: 路由守卫
系统 SHALL 对需要认证的路由进行守卫，未登录用户访问时自动跳转到登录页；非 admin 用户访问管理页面时显示 403 提示。

#### Scenario: 未登录用户访问认证页面
- **WHEN** 未登录用户访问 `/` 或 `/tickets/:id` 等需要认证的页面
- **THEN** 系统自动跳转到 `/login`，登录后回到原始请求页面

#### Scenario: 非 admin 访问管理页面
- **WHEN** role 为 "user" 的用户访问 `/admin/*` 路径
- **THEN** 页面显示"需要管理员权限"提示

### Requirement: 登出
系统 SHALL 在顶栏显示当前用户名和退出按钮，点击退出后清除认证信息并跳转到登录页。

#### Scenario: 用户点击退出
- **WHEN** 用户点击顶栏的"退出"按钮
- **THEN** 系统清除 localStorage 中的 token 和用户信息，跳转到登录页
