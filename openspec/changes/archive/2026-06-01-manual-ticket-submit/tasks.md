## 1. 后端：桥接提交端点

- [x] 1.1 在 `HumanHandler` 新增 `Submit` 方法：解析 `api_key_id` + `project` + `name` + `args`，校验 API Key 存在且启用，调用 `TicketService.Submit`
- [x] 1.2 在 `router.go` 注册 `POST /tickets/submit` 路由，挂载 JWT 认证中间件

## 2. 前端：提交对话框

- [x] 2.1 在 `api.ts` 新增 `submitTicket` 方法，调用 `POST /tickets/submit`
- [x] 2.2 在 `TicketListPage` 新增"提交工单"按钮和 Dialog 组件：包含 API Key 下拉、程序下拉、参数 textarea
- [x] 2.3 Dialog 打开时加载 API Keys 和 Programs 列表填充下拉选项
- [x] 2.4 提交成功后关闭 Dialog 并刷新工单列表；失败时在 Dialog 内显示错误信息
