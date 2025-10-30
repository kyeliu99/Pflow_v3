# PFlow v3

PFlow 是一个面向复杂业务流程的可视化流程搭建工具与工单管理平台。平台以“流程引擎 + 工单系统”的双能力中心设计，支持表单构建、流程编排、工单生命周期管理、权限与协同，同时提供开放的 OpenAPI 与 npm SDK，满足二次开发与深度集成诉求。

## 架构概览

> 所有模块均为独立微服务，便于弹性扩展与独立部署。

```
apps/
  frontend/           # React + Vite 控制台，封装 npm SDK 的演示入口
libs/
  shared/             # Go 共享库：配置、数据库、消息队列、HTTP、观测等
services/
  gateway/            # API 聚合 & BFF，统一认证、路由、OpenAPI 暴露点
  identity/           # 身份与权限管理，面向多角色的 RBAC 能力
  form/               # 拖拽式表单模型存储与版本管理
  workflow/           # Camunda 8 (Zeebe) 流程编排适配层
  ticket/             # 工单调度、状态同步、事件消费
```

技术选型遵循“全部基于成熟开源生态”：

- **流程引擎**：Camunda 8 / Zeebe 客户端 (`github.com/camunda/zeebe`)
- **编程语言**：Go 1.21 (微服务)、TypeScript + React 18 (前端)
- **通信协议**：HTTP/JSON + Kafka 事件流 (`github.com/segmentio/kafka-go`)
- **数据持久化**：PostgreSQL (`gorm.io/gorm` + `gorm.io/driver/postgres`)
- **配置与观测**：`github.com/joho/godotenv`、Prometheus 客户端 (`github.com/prometheus/client_golang`)

## 本地开发与调试

### 1. 依赖准备

- Go 1.21+
- Node.js 18+
- Docker & Docker Compose (用于启动数据库、Kafka、Camunda)

### 2. 启动基础设施

项目根目录提供 `docker-compose.yml`（见下文示例）用于拉起依赖服务。

```bash
docker compose up -d postgres kafka camunda
```

- PostgreSQL 暴露在 `5432`
- Kafka 暴露在 `9092`
- Camunda/Zeebe 网关暴露在 `8088`

### 3. 配置环境变量

复制 `.env.example` 到各服务根目录或以环境变量形式注入：

```
SERVICE_NAME=gateway
HTTP_PORT=8080
POSTGRES_DSN=postgres://pflow:pflow@localhost:5432/pflow?sslmode=disable
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=pflow-events
CAMUNDA_URL=0.0.0.0:26500
```

### 4. 启动微服务

每个服务独立运行，示例命令：

```bash
# Gateway
cd services/gateway && go run ./cmd/main.go

# 表单服务
cd services/form && go run ./cmd/main.go

# 工单服务
cd services/ticket && go run ./cmd/main.go

# 身份服务
cd services/identity && go run ./cmd/main.go

# 工作流服务（Camunda）
cd services/workflow && go run ./cmd/main.go
```

可借助 `air`, `fresh` 等热加载工具提升体验。

### 5. 前端控制台

```bash
cd apps/frontend
npm install
npm run dev
```

前端默认代理 `/api` 到 `http://localhost:8080`，可在 `vite.config.ts` 调整。

## OpenAPI 与 SDK

- Gateway 统一暴露 REST API，后续可整合 `swagger`/`openapi` 生成器。
- `apps/frontend/src/lib/api.ts` 提供 axios 封装示例。
- 可在 npm 包中导出 React hooks（例如 `useForms`, `useTickets`）进一步封装。

## Docker Compose 示例

以下 compose 片段演示如何在本地拉起依赖组件：

```yaml
version: "3.9"
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: pflow
      POSTGRES_PASSWORD: pflow
      POSTGRES_DB: pflow
    ports:
      - "5432:5432"
  kafka:
    image: bitnami/kafka:3.6.1
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_LISTENERS: PLAINTEXT://:9092
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"
  zookeeper:
    image: bitnami/zookeeper:3.9.2
    environment:
      ALLOW_ANONYMOUS_LOGIN: "yes"
    ports:
      - "2181:2181"
  camunda:
    image: camunda/zeebe:8.3.0
    environment:
      ZEEBE_LOG_LEVEL: info
    ports:
      - "8088:8080"
      - "26500:26500"
```

> 可根据需要扩展 compose 以包含 Jaeger、Prometheus 等观测组件。

## 目录内说明

- `libs/shared`: 统一的配置加载、数据库/消息队列连接、HTTP Server 封装、Prometheus Metrics 注册。
- `services/*`: 每个服务都使用共享库，以清晰的领域边界组织。
- `apps/frontend`: React 控制台示例，提供流程编排与工单看板的可视化界面。

## 下一步规划

1. **领域模型持久化**：基于 GORM 定义 Form/Ticket/User 等实体与迁移。
2. **OpenAPI & 文档化**：在 Gateway 集成 `swaggo/gin-swagger` 自动生成接口文档。
3. **事件驱动编排**：将 Camunda 任务事件写入 Kafka，Ticket Service 实时更新状态。
4. **权限系统**：Identity Service 提供 JWT/OIDC 集成与多租户支持。
5. **前端增强**：接入 React Flow / low-code 编排组件，实现真实拖拽能力。

该仓库提供的骨架代码全部基于开源生态，便于在其上快速迭代业务能力。
