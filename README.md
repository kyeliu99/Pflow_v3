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

- 默认 compose 会一次性拉起 PostgreSQL、Zookeeper、Kafka 与 Camunda：

```bash
docker compose up -d postgres zookeeper kafka camunda
```

- 首次启动 PostgreSQL 会自动执行 `scripts/postgres/init.sql`，确保创建 `pflow` 数据库与登录角色。
- 如果此前已经启动过旧版本的容器导致卷内缺少该角色，可执行 `docker compose down -v postgres` 清理卷后再启动，或手动进入容器执行 `psql -U postgres -c "CREATE ROLE pflow LOGIN PASSWORD 'pflow';"` 与 `psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE pflow TO pflow;"`。

- PostgreSQL 暴露在 `5432`
- Kafka 暴露在 `9092`（容器互联 `kafka:9092`，宿主机备用监听 `localhost:9092`）
- Camunda/Zeebe 网关暴露在 `26500`（gRPC）与 `8088`（控制台）
- 如果某个容器启动失败，可通过 `docker compose logs <service>` 查看原因

### 3. 配置环境变量

将示例配置复制为仓库根目录的 `.env`（一次即可）：

```bash
cp .env.example .env
```

所有微服务都会自动读取仓库根目录的 `.env`、`.env.local` 以及 `.env.d/*.env` 文件，无需再为每个服务重复拷贝。

> `.env.example` 不再预设统一的 `HTTP_PORT`，各服务会在未显式设置时使用推荐端口（Gateway=8080、Form=8081、Identity=8082、Ticket=8083、Workflow=8084）。如需修改，请在运行命令前通过环境变量覆盖，例如 `HTTP_PORT=9000 go run ./cmd/main.go`。

如需加载额外的配置文件，可通过 `PFLOW_ENV_FILES` 指定逗号分隔的路径列表。

> `.env` 中的 `POSTGRES_IMAGE`、`ZOOKEEPER_IMAGE`、`KAFKA_IMAGE`、`CAMUNDA_IMAGE` 变量可按需指向企业私有仓库或镜像加速服务，以避免 Docker Hub 拉取受限。

### 4. 启动微服务

建议在独立终端中分别启动各个服务（默认端口见下表，可按需覆盖 `HTTP_PORT`）：

| 服务 | 目录 | 默认端口 | 启动命令 |
| --- | --- | --- | --- |
| API Gateway | `services/gateway` | 8080 | `go run ./cmd/main.go` |
| Form Service | `services/form` | 8081 | `go run ./cmd/main.go` |
| Identity Service | `services/identity` | 8082 | `go run ./cmd/main.go` |
| Ticket Service | `services/ticket` | 8083 | `go run ./cmd/main.go` |
| Workflow Service | `services/workflow` | 8084 | `go run ./cmd/main.go` |

启动顺序建议为：先运行依赖基础设施与 API Gateway，再依次启动领域服务。可借助 `air`、`fresh` 等热加载工具提升开发效率。

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

以下 compose 片段演示如何在本地拉起依赖组件（镜像名称支持通过根目录 `.env` 中的 `POSTGRES_IMAGE`/`ZOOKEEPER_IMAGE`/`KAFKA_IMAGE`/`CAMUNDA_IMAGE` 覆盖，便于切换到私有仓库或镜像加速源）：

```yaml
version: "3.9"
services:
  postgres:
    image: ${POSTGRES_IMAGE:-postgres:16}
    environment:
      POSTGRES_USER: pflow
      POSTGRES_PASSWORD: pflow
      POSTGRES_DB: pflow
    ports:
      - "5432:5432"
  zookeeper:
    image: ${ZOOKEEPER_IMAGE:-bitnami/zookeeper:3.9}
    environment:
      ALLOW_ANONYMOUS_LOGIN: "yes"
    ports:
      - "2181:2181"
  kafka:
    image: ${KAFKA_IMAGE:-bitnami/kafka:3.7}
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT
      KAFKA_CFG_LISTENERS: PLAINTEXT://:9092,PLAINTEXT_HOST://:29092
      KAFKA_CFG_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092,PLAINTEXT_HOST://localhost:9092
      KAFKA_CFG_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE: "true"
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
    depends_on:
      - zookeeper
    ports:
      - "9092:9092"
      - "29092:29092"
  camunda:
    image: ${CAMUNDA_IMAGE:-camunda/zeebe:8.3.0}
    environment:
      ZEEBE_LOG_LEVEL: info
      ZEEBE_GATEWAY_NETWORK_HOST: 0.0.0.0
    ports:
      - "26500:26500"
      - "8088:8080"
```

> 可根据需要扩展 compose 以包含 Jaeger、Prometheus 等观测组件。

### 解决镜像拉取超时/失败

- **优先检查网络**：错误 `Client.Timeout exceeded while awaiting headers` 通常意味着无法连接 Docker Hub。可先尝试 `docker pull ${ZOOKEEPER_IMAGE}` 验证网络连通性。
- **使用预拉取脚本**：执行 `./scripts/docker/pull-dependencies.sh` 会按 `.env` 或默认值提前拉取依赖镜像，并在补丁标签不存在时自动回退到次版本号标签（如 `3.9.1 -> 3.9`）。脚本运行成功后再执行 `docker compose up -d postgres zookeeper kafka camunda` 可显著降低首次启动失败的概率。
- **开箱即用的镜像加速 compose 文件**：仓库提供 `docker-compose.mirror.yml`，预置了 [DaoCloud 镜像服务](https://docker.m.daocloud.io) 的镜像地址，可直接配合基础 compose 文件使用：

  ```bash
  docker compose -f docker-compose.yml -f docker-compose.mirror.yml pull
  docker compose -f docker-compose.yml -f docker-compose.mirror.yml up -d
  ```

  如仍需切换到企业内部仓库，可在执行命令前设置环境变量（例如 `POSTGRES_IMAGE`），该覆盖文件同样会读取这些变量。
- **使用镜像加速器**：在 `~/.docker/config.json` 中增加 `"registry-mirrors": ["https://registry.docker-cn.com", "https://<你的镜像服务域名>"]`，或使用企业内网镜像仓库。
- **覆盖镜像地址**：根据 `.env.example` 添加 `ZOOKEEPER_IMAGE=<your-registry>/bitnami/zookeeper:3.9` 等变量，重新执行 `docker compose up -d` 即可改用自定义仓库。
- **手动预拉取**：对网络较慢的环境，可提前运行 `docker pull` 将所需镜像拉取到本地，再执行 compose。
- **确认镜像标签是否存在**：Bitnami 会定期下线旧补丁版本（例如 `bitnami/kafka:3.6.1`）。为降低风险，本仓库默认使用带次版本号的长期标签（如 `bitnami/zookeeper:3.9`、`bitnami/kafka:3.7`）。你可以在启动前运行 `docker manifest inspect <image>` 或访问镜像仓库标签页确认可用版本，再在 `.env` 中调整 `*_IMAGE` 变量。

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
