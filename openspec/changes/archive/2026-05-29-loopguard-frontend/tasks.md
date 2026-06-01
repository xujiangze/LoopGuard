## 1. 项目脚手架

- [x] 1.1 创建 Vite + React + TypeScript 项目到 `web/` 目录（`pnpm create vite web --template react-ts`）
- [x] 1.2 安装依赖：react-router-dom、tailwindcss、@tailwindcss/vite
- [x] 1.3 配置 Tailwind CSS（tailwind.config.ts + index.css 引入）
- [x] 1.4 安装 shadcn/ui（`pnpm dlx shadcn@latest init`，选 New York 风格、neutral 色调）
- [x] 1.5 配置 vite.config.ts 中的 proxy 或确认 VITE_API_URL 环境变量方案
- [x] 1.6 验证 `pnpm dev` 启动成功

## 2. 后端 CORS 支持

- [x] 2.1 给 Go 后端添加 `github.com/gin-contrib/cors` 依赖
- [x] 2.2 在 `internal/api/router.go` 的 NewRouter 中注册 CORS 中间件（允许前端 origin、支持 Authorization header）
- [x] 2.3 验证前后端跨域请求正常

## 3. 基础设施（认证 + 路由 + 布局）

- [x] 3.1 创建 `src/types/index.ts`：后端模型 TS 类型定义（User、Ticket、Program、APIKey、TicketStatus 等）
- [x] 3.2 创建 `src/lib/auth.ts`：JWT 存取（getToken/setToken/clearToken）、用户信息存取
- [x] 3.3 创建 `src/lib/api.ts`：封装 fetch（baseURL、自动 Bearer header、401 自动跳登录、统一错误处理）
- [x] 3.4 创建 `src/hooks/useAuth.ts`：AuthContext + useAuth hook（提供 user/role/login/logout）
- [x] 3.5 创建 `src/App.tsx`：react-router 路由表（/login、/、/tickets/:id、/admin/programs、/admin/users、/admin/api-keys）
- [x] 3.6 创建 `src/components/ProtectedRoute.tsx`：JWT 守卫，未登录跳 /login
- [x] 3.7 创建 `src/components/AdminRoute.tsx`：admin 角色守卫，非 admin 显示 403
- [x] 3.8 创建 `src/components/Layout.tsx`：全局布局（侧边栏导航 + 顶栏用户信息/退出），admin 才显示管理菜单分组

## 4. 登录页

- [x] 4.1 创建 `src/pages/LoginPage.tsx`：居中卡片、用户名+密码输入、登录按钮
- [x] 4.2 调用 `POST /api/v1/auth/login`，成功存 JWT 跳首页，失败显示错误提示
- [x] 4.3 验证：未登录访问首页自动跳登录页，登录成功跳回

## 5. 工单列表页（首页）

- [x] 5.1 创建 `src/pages/TicketListPage.tsx`：页面标题 + 状态筛选 Tabs（全部/待审批/已完成/已驳回/执行失败/dryrun失败）
- [x] 5.2 调用 `GET /api/v1/tickets?status=xxx` 获取工单列表，展示表格（ID、程序、状态标签、提交时间、查看按钮）
- [x] 5.3 状态标签带颜色区分（待审批=黄、已完成=绿、已驳回=红、执行失败=红、dryrun失败=灰）
- [x] 5.4 空列表展示"暂无工单"空状态
- [x] 5.5 验证：登录后默认进入列表页，Tab 切换筛选正常

## 6. 工单详情页

- [x] 6.1 创建 `src/pages/TicketDetailPage.tsx`：基本信息区（程序名、状态、提交时间）
- [x] 6.2 AI 提交参数区（JSON 格式化展示）
- [x] 6.3 Dry-run 输出区（代码块展示，保留原始格式）
- [x] 6.4 审批操作区：待审批状态显示"批准执行"和"驳回"按钮，其他状态隐藏
- [x] 6.5 批准操作：调用 `POST /approve`，成功跳回列表
- [x] 6.6 驳回操作：弹 Dialog 输入原因，调用 `POST /reject`，成功跳回列表
- [x] 6.7 已完成工单展示执行结果区（退出码、耗时、stdout/stderr）
- [x] 6.8 已驳回工单展示驳回原因
- [x] 6.9 工单不存在时展示提示 + 返回链接
- [x] 6.10 验证：从列表点击查看→详情页展示正确→审批/驳回→跳回列表

## 7. 程序管理页（Admin）

- [x] 7.1 创建 `src/pages/ProgramPage.tsx`：程序列表表格（项目/名称、二进制路径、审批人、启用状态、编辑按钮）
- [x] 7.2 注册新程序弹窗表单（项目名、程序名、二进制路径、审批人下拉选择、超时、参数白名单 JSON）
- [x] 7.3 调用 `POST /api/v1/programs` 注册，成功刷新列表
- [x] 7.4 编辑弹窗：修改启用/禁用状态、审批人、超时，调用 `PUT /api/v1/programs/:id`
- [x] 7.5 审批人下拉数据来源：目前后端无用户列表 API，暂用文本输入 ID 或后续补充
- [x] 7.6 验证：注册新程序→列表刷新→编辑启用/禁用

## 8. 用户管理页（Admin）

- [x] 8.1 创建 `src/pages/UserPage.tsx`：用户列表表格（ID、用户名、角色、创建时间）
- [x] 8.2 创建用户弹窗表单（用户名、密码、角色选择 user/admin）
- [x] 8.3 调用 `POST /api/v1/users` 创建，成功刷新列表，失败显示错误
- [x] 8.4 验证：创建新用户→列表刷新

## 9. API Key 管理页（Admin）

- [x] 9.1 创建 `src/pages/ApiKeyPage.tsx`：Key 列表表格（ID、名称、启用状态、创建时间）
- [x] 9.2 创建 Key 弹窗表单（名称输入框）
- [x] 9.3 调用 `POST /api/v1/api-keys` 创建
- [x] 9.4 创建成功后弹出 Alert Dialog：警告文案 + 明文 Key 展示 + 复制按钮 + "我已保存，关闭"按钮
- [x] 9.5 复制按钮调用 `navigator.clipboard.writeText`，复制后按钮变为"已复制"
- [x] 9.6 验证：创建 Key→弹窗展示明文→复制→关闭→列表刷新

## 10. 收尾与验证

- [x] 10.1 全页面走查：登录→列表→详情审批→管理页操作，无报错
- [x] 10.2 `pnpm build` 验证生产构建无错误
- [x] 10.3 检查响应式：桌面端 1280px+ 布局正常
- [x] 10.4 Commit：`git add web/ && git commit -m "feat: LoopGuard 前端（React + shadcn/ui）"`
