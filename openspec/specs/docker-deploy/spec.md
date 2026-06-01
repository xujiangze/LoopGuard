# Docker Deploy

## Purpose

提供 Docker 容器化部署方案，支持一键启动 LoopGuard 服务及依赖数据库。

## Requirements

### Requirement: Multi-stage Dockerfile
系统 SHALL 提供多阶段 Dockerfile，依次构建前端（Node.js）、编译 Go 二进制（嵌入前端）、生成最小运行时镜像。

#### Scenario: Docker 构建成功
- **WHEN** 执行 `docker build -t loopguard .`
- **THEN** 构建成功，最终镜像大小不超过 50M（Alpine 基础）

#### Scenario: 容器启动服务
- **WHEN** 执行 `docker run loopguard serve` 并提供必要环境变量
- **THEN** 服务在容器内 :8080 启动，API 和前端页面均可访问

### Requirement: Docker Compose 编排
系统 SHALL 提供 docker-compose.yaml，定义 loopguard 和 mysql 两个服务，支持一键部署。

#### Scenario: 一键启动
- **WHEN** 执行 `docker compose up -d`
- **THEN** MySQL 和 loopguard 容器均启动，loopguard 等待 MySQL 就绪后开始服务

#### Scenario: 数据持久化
- **WHEN** MySQL 容器重启或重建
- **THEN** 通过 named volume 持久化的数据仍然存在

### Requirement: 内存限制配置
docker-compose.yaml SHALL 为每个服务配置内存限制：MySQL 1G，loopguard app 4G。

#### Scenario: 内存超限
- **WHEN** 某个服务内存使用超过配置的限制
- **THEN** 容器被 OOM kill 而不影响另一个服务

### Requirement: 环境变量模板
项目 SHALL 提供 .env.example 文件，包含所有 LOOPGUARD_ 环境变量及注释说明。

#### Scenario: 新用户部署
- **WHEN** 用户复制 .env.example 为 .env 并填写必要值
- **THEN** `docker compose up -d` 可正常启动
