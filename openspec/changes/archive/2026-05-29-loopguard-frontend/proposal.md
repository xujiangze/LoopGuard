## Why

LoopGuard 后端已完成实现，提供完整的审批工单 API（AI 提交、人工审批、管理接口），但目前缺少前端界面。审批人需要通过浏览器查看待审批工单、审查 dry-run 输出、执行审批/驳回操作；管理员需要注册程序、管理用户和 API Key。现在需要实现 React 前端 SPA 来对接后端 API，让审批和管理流程可视化可用。

## What Changes

- 新增 React + TypeScript + Vite 前端项目（`web/` 目录）
- 实现登录页，对接 `POST /api/v1/auth/login` 获取 JWT
- 实现工单列表页（首页），展示当前审批人的待办工单，支持按状态筛选
- 实现工单详情页，展示程序信息、AI 参数、dry-run 输出，提供审批/驳回操作
- 实现管理页面（admin only）：程序管理、用户管理、API Key 管理
- 实现全局布局（侧边栏导航 + 顶栏用户信息）
- 实现 JWT 认证守卫和角色权限控制（user / admin）

## Capabilities

### New Capabilities

- `auth-flow`: JWT 登录/登出/守卫，token 持久化，401 自动跳登录
- `ticket-approval`: 工单列表筛选 + 工单详情查看 + 审批/驳回交互
- `admin-panel`: 程序注册/管理、用户创建、API Key 创建（含明文一次性展示）

### Modified Capabilities

（无，这是全新前端，不修改后端能力）

## Impact

- 新增 `web/` 目录，包含完整 React 项目
- 前端独立部署，通过 CORS 与后端通信（后端需配置 CORS 中间件，当前未实现）
- 后端需要新增 CORS 配置以允许前端跨域请求
- 依赖：React 18、react-router v6、Tailwind CSS、shadcn/ui、Vite
