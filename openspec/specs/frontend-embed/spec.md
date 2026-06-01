# Frontend Embed

## Purpose

将前端构建产物通过 `//go:embed` 嵌入 Go 二进制，实现单一可执行文件部署，简化运维。

## Requirements

### Requirement: 前端静态文件嵌入 Go 二进制
系统 SHALL 在编译时通过 `//go:embed` 将前端构建产物（web/dist/）嵌入 Go 二进制。

#### Scenario: 无 dist 目录时编译失败
- **WHEN** web/dist/ 目录不存在或为空时执行 `go build`
- **THEN** 编译报错，提示前端构建产物缺失

#### Scenario: 嵌入后二进制正常服务前端
- **WHEN** Go 二进制启动并监听端口
- **THEN** 访问根路径 `/` 返回 index.html，`/assets/*` 返回对应静态资源

### Requirement: Gin 静态文件路由
系统 SHALL 在 Gin 路由中注册前端静态文件服务，路径规则：`/assets/*` → 嵌入的静态资源文件，`/` → index.html。

#### Scenario: 访问前端页面
- **WHEN** 浏览器请求 `GET /`
- **THEN** 返回嵌入的 index.html，Content-Type 为 text/html

#### Scenario: 访问静态资源
- **WHEN** 浏览器请求 `GET /assets/main-abc.js`
- **THEN** 返回对应的 JS 文件，Content-Type 为 application/javascript

#### Scenario: API 路由不受影响
- **WHEN** 请求 `GET /api/v1/tickets`
- **THEN** 走原有 API 路由逻辑，不受静态文件服务影响
