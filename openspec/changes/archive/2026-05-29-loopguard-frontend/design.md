## Context

LoopGuard 后端已完成，提供三组 HTTP API（AI 接口、人工接口、管理接口），全部基于 JSON。当前没有任何前端界面，审批人和管理员只能通过 curl 或 API 工具操作。需要构建 React SPA 作为审批和管理的可视化界面。

后端 API 基地址通过 `VITE_API_URL` 环境变量配置。后端当前未配置 CORS 中间件，需在实现前先为后端添加 CORS 支持。

## Goals / Non-Goals

**Goals:**
- 审批人可通过浏览器登录、查看待审批工单、审批/驳回
- 管理员可通过浏览器注册程序、创建用户、创建 API Key
- 前端独立部署，通过 CORS 与后端通信
- 响应式设计，适配桌面端使用（主要场景）

**Non-Goals:**
- 移动端适配（审批人主要在桌面工作）
- 国际化（仅中文界面）
- 实时刷新 / WebSocket（审批后直接跳回列表）
- 离线支持 / PWA
- 前端单元测试（页面少、交互简单，后端已有完整测试覆盖）

## Decisions

### D1: React + TypeScript + Vite

**选择**: Vite 作为构建工具，React 18 + TypeScript。

**替代方案**: Next.js（SSR 不需要，增加复杂度）、CRA（已废弃）。

**理由**: SPA 场景，Vite 开发体验最好，冷启动快。TypeScript 保证类型安全，后端模型有清晰的 TS 类型映射。

### D2: Tailwind CSS + shadcn/ui

**选择**: Tailwind CSS 做样式基础，shadcn/ui 提供 UI 组件。

**替代方案**: Ant Design（开箱即用但风格固定，包体积大）、MUI（同理）。

**理由**: shadcn/ui 可定制性强，复制到项目中完全可控。Tailwind 与 shadcn/ui 天然配合。LoopGuard 页面少，不需要庞大的组件库。

### D3: react-router v6 做路由

**选择**: react-router v6，HashRouter 模式。

**替代方案**: TanStack Router（更现代但学习成本高，项目规模不需要）。

**理由**: react-router 生态成熟，v6 API 简洁。用 HashRouter 避免后端 SPA fallback 配置。

### D4: React Context + useState 管理认证状态

**选择**: useAuth Context 管理登录状态，不做全局状态管理库。

**替代方案**: Zustand / Redux（过度工程化）。

**理由**: 全局状态只有 JWT token 和用户角色，React Context 完全够用。其他页面数据通过组件内 useState + useEffect 获取。

### D5: JWT 存 localStorage

**选择**: JWT 存入 localStorage，每次请求从 localStorage 读取。

**替代方案**: HttpOnly Cookie（需要后端配合改造）。

**理由**: 后端已设计为 Bearer token 模式，前端独立部署跨域场景下 localStorage 最直接。XSS 风险通过 CSP 和输入转义缓解。

### D6: fetch 做请求，不引入 axios

**选择**: 原生 fetch + 统一封装。

**替代方案**: axios（API 请求少，没必要增加依赖）。

**理由**: 后端 API 全部返回 JSON，没有复杂的请求拦截需求。一个 `api.ts` 封装 baseURL + auth header + 401 处理即可。

### D7: 目录结构按页面组织

**选择**: `pages/` 按页面分文件，`components/` 放共享组件，`lib/` 放工具函数。

**理由**: 6 个页面，规模小，flat 结构最清晰。不需要 features/ 或 modules/ 等深层组织。

## Risks / Trade-offs

**[后端缺少 CORS]** → 实现前端前，需先给 Go/Gin 后端添加 CORS 中间件（`github.com/gin-contrib/cors`）。这是前置依赖。

**[JWT 过期无刷新机制]** → 后端 JWT 有效期 12 小时。过期后前端 401 跳登录页重新登录即可，第一期不做 refresh token。

**[API Key 明文只显示一次]** → 创建 API Key 后用 Alert Dialog 强提示 + 复制按钮，关闭弹窗后无法再看到明文。UI 上要足够醒目。

**[shadcn/ui 组件按需安装]** → shadcn/ui 不是 npm 包，需要 `npx shadcn-ui@latest add <component>` 逐个安装。实现时按需添加，避免一次性引入太多。
