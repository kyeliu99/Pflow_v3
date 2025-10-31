# PFlow v3

PFlow 是一个面向复杂业务流程的可视化流程搭建工具与工单管理平台。平台以“流程引擎 + 工单系统”的双能力中心设计，支持表单构建、流程编排、工单生命周期管理、权限与协同，同时提供开放的 OpenAPI 与 npm SDK，满足二次开发与深度集成诉求。

## 架构概览

> 所有模块均为独立微服务，便于弹性扩展与独立部署。

```
apps/
  frontend/           # React + Vite 控制台，封装 npm SDK 的演示入口
libs/
  shared/             # Go 共享库：配置、数据库、消息队列、HTTP、观测等
  components/         # 可复用的领域服务组件（表单、身份、工单、流程）
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

## 可复用领域组件

为提升可复用性，所有领域逻辑均沉淀在 `libs/components` 模块中，遵循“领域模型 + 仓储接口 + HTTP Handler”三层结构：

- `components/form`、`components/identity`、`components/ticket`、`components/workflow` 均导出 GORM 模型、仓储实现与基于 chi 的路由注册器。
- 每个 Handler 均提供 `Mount(router, basePath)` 方法，可在任意 Go 服务中按需挂载，默认路径分别为 `/forms`、`/users`、`/tickets` 与 `/workflows`。
- 若需要自定义存储，可实现对应的 `Repository` 接口并传入 `NewHandler`，领域层无需修改。

针对高并发场景，`components/ticket` 还额外提供：

- `TicketSubmission` 模型与 `SubmissionRepository`：持久化异步请求、统计队列指标。
- `QueueCoordinator`：负责写入数据库、发布 Kafka 消息，并通过 `SubmissionCoordinator` 接口对外暴露。
- `QueueWorker`：消费 Kafka 消息、调用仓储落地工单，可通过 `mq.NewConsumer` 快速接入任意服务。

消息队列的底层封装在 `libs/shared/mq` 中，基于 `github.com/segmentio/kafka-go` 提供 `NewProducer`、`NewConsumer` 等主流 API，避免重复配置 Dialer、重试与客户端标识。任何服务只需在配置中提供 `*_KAFKA_BROKERS`、`*_QUEUE_TOPIC` 与 `*_QUEUE_GROUP` 即可复用同一套组件。

示例（在自定义服务中复用工单组件）：

```go
import (
    ticketcmp "github.com/pflow/components/ticket"
    "github.com/go-chi/chi/v5"
    "gorm.io/gorm"
)

func wire(db *gorm.DB, router chi.Router) {
    repo := ticketcmp.NewGormRepository(db)
    handler := ticketcmp.NewHandler(repo)
    handler.Mount(router, "/api/tickets")
}
```

Gateway 及各领域微服务即是通过上述组件拼装而成，这意味着同一套业务能力可以被二次包装为内部 RPC 服务、任务处理器或按需暴露为新的 API。

## 本地开发与调试

为提升可复用性，所有领域逻辑均沉淀在 `libs/components` 模块中，遵循“领域模型 + 仓储接口 + HTTP Handler”三层结构：

- Go 1.21+
- Node.js 18+
- PostgreSQL 15+（或以上版本）
- `psql`、`initdb`、`pg_ctl` 等 PostgreSQL 命令行工具
- OpenJDK 17+（用于运行 Kafka 与 Camunda Zeebe）
- `curl`、`tar`、`uuidgen`（用于下载与解压官方发行版）

### 2. 初始化 PostgreSQL（本地进程）

`scripts/postgres/run-local.sh` 基于官方工具链创建并管理独立的数据目录，无需 Docker：

```bash
# 第一次启动会在 .data/postgres 下初始化数据，并监听 127.0.0.1:5432
./scripts/postgres/run-local.sh start

# 查看运行状态
./scripts/postgres/run-local.sh status

# 结束进程（例如切换分支或释放端口时）
./scripts/postgres/run-local.sh stop
```

脚本会自动向 `postgresql.conf` 写入监听地址、端口与连接数量，同时在 `pg_hba.conf` 中启用本地密码登录。若系统尚未安装 PostgreSQL，可通过以下方式获取官方发行版：

- macOS（Homebrew）：`brew install postgresql@16`
- Debian/Ubuntu：`sudo apt install postgresql postgresql-contrib`
- Windows：下载 [postgresql.org](https://www.postgresql.org/download/) 提供的安装包，并在 Git Bash / WSL 中执行脚本。

常用覆盖项：

- `POSTGRES_PORT`：监听端口（默认 5432）
- `POSTGRES_SUPERUSER` / `POSTGRES_PASSWORD`：初始化的超级用户（默认 postgres/postgres）
- `PFLOW_POSTGRES_DATA_DIR`：数据目录（默认 `.data/postgres`）
- `PFLOW_POSTGRES_SKIP_BOOTSTRAP=1`：跳过自动执行 `scripts/postgres/bootstrap.sh`

若已有 PostgreSQL 实例，只需执行 bootstrap 脚本即可幂等创建 `pflow` 角色与数据库：

```bash
export PGPASSWORD=<postgres 密码>
POSTGRES_HOST=<你的地址> POSTGRES_PORT=<端口> ./scripts/postgres/bootstrap.sh
```

脚本内部复用 `scripts/postgres/init.sql`，也可手动执行以下 SQL：

```sql
CREATE ROLE pflow LOGIN PASSWORD 'pflow';
CREATE DATABASE pflow OWNER pflow;
```

### 3. 启动 Kafka（KRaft 单节点）

`scripts/kafka/run-local.sh` 会下载 Apache Kafka 官方二进制并以 KRaft 模式启动单节点集群：

```bash
# 首次执行会在 .data/kafka 下完成下载与初始化
./scripts/kafka/run-local.sh start

# 确认服务是否就绪
./scripts/kafka/run-local.sh status

# 停止本地 Kafka
./scripts/kafka/run-local.sh stop
```

脚本需要 `java`、`curl`、`tar` 与 `uuidgen` 命令，默认监听 `localhost:9092`，支持以下可选变量：

- `KAFKA_PORT` / `KAFKA_CONTROLLER_PORT`
- `PFLOW_KAFKA_DATA_DIR`
- `KAFKA_VERSION` / `KAFKA_SCALA_VERSION`

### 4. 启动 Camunda Zeebe（可选）

流程服务默认以内置仓储运行，若需与 Camunda 8 集成，可通过 `scripts/camunda/run-local.sh` 下载并启动 Zeebe：

```bash
./scripts/camunda/run-local.sh start
./scripts/camunda/run-local.sh status
./scripts/camunda/run-local.sh stop
```

脚本同样依赖 `java`、`curl` 与 `tar`，默认网关监听 `localhost:26500`。如使用 Camunda SaaS，可直接在 `.env` 中配置远程 `CAMUNDA_URL`，无需启动本地实例。

### 5. 配置环境变量

将示例配置复制为仓库根目录的 `.env`（一次即可）：

```bash
cp .env.example .env
```

所有微服务都会自动读取仓库根目录的 `.env`、`.env.local` 以及 `.env.d/*.env` 文件，无需再为每个服务重复拷贝。

> `.env.example` 提供 `FORM_DATABASE_DSN`/`FORM_HTTP_PORT` 等服务级变量：若未配置则自动回退到 `POSTGRES_DSN` 与推荐端口（Gateway=8080、Form=8081、Identity=8082、Ticket=8083、Workflow=8084），也可以继续通过全局 `HTTP_PORT`/`POSTGRES_DSN` 快速覆盖。运行命令前按需设置对应环境变量即可，例如 `FORM_HTTP_PORT=9001 go run ./cmd/main.go`。

如需加载额外的配置文件，可通过 `PFLOW_ENV_FILES` 指定逗号分隔的路径列表。

### 6. 启动微服务

建议在独立终端中分别启动各个服务（默认端口见下表，可按需覆盖 `HTTP_PORT`）：

| 服务 | 目录 | 默认端口 | 启动命令 |
| --- | --- | --- | --- |
| API Gateway | `services/gateway` | 8080 | `go run ./cmd/main.go` |
| Form Service | `services/form` | 8081 | `go run ./cmd/main.go` |
| Identity Service | `services/identity` | 8082 | `go run ./cmd/main.go` |
| Ticket Service | `services/ticket` | 8083 | `go run ./cmd/main.go` |
| Ticket Worker（队列消费者） | `services/ticket` | - | `go run ./cmd/worker/main.go` |
| Workflow Service | `services/workflow` | 8084 | `go run ./cmd/main.go` |

> 服务在启动时会调用 `cfg.DatabaseDSN(<service>)` 与 `cfg.ResolveServiceHTTPPort(<service>, <fallback>)`：只需在 `.env` 或运行命令前设置 `FORM_DATABASE_DSN`、`TICKET_HTTP_PORT` 等变量即可让组件无缝连接不同的数据库实例或监听端口。`libs/shared/database.ConnectWithDSN` 会缓存命名连接，便于在同一进程中复用多个数据源。队列消费者同时读取 `TICKET_QUEUE_TOPIC`、`TICKET_QUEUE_GROUP` 等变量，并通过 `libs/shared/mq` 连接 Kafka。

启动顺序建议为：先运行依赖基础设施与 API Gateway，再依次启动领域服务。可借助 `air`、`fresh` 等热加载工具提升开发效率。

### 7. 前端控制台

```bash
cd apps/frontend
# 验证代码可正常打包（无语法错误）
npm run build

技术选型评估

- **单语言栈**：所有后端微服务均以 Go 实现，依赖 `chi`、`gorm` 等主流社区项目，降低跨语言维护成本。
- **组件化复用**：核心领域逻辑沉淀在 `libs/components`，通过 Handler + Repository 组合实现“即插即用”，适合拆装成 BFF、RPC 或后台任务。
- **可靠的消息队列**：基于 `github.com/segmentio/kafka-go` 的 `libs/shared/mq` 统一封装生产者 / 消费者，结合 `TicketSubmission` 模型实现可重放的异步工单管道。
- **前端防呆体验**：React 控制台利用队列指标与提交状态轮询，防止重复点击并即时反馈后台进度。
- **无个人依赖**：所有三方库均来自活跃的官方 / 社区组织，便于企业内网镜像与安全评估。
API 约定
所有接口通过 Gateway 统一访问（前缀 /api），核心接口：
服务
接口路径与功能
表单服务
GET/POST/PUT/DELETE /api/forms/（表单 CRUD）
身份服务
GET/POST /api/users/（用户管理）、GET /api/roles/（角色查询）
工单服务
POST /api/tickets/submissions/（异步创建工单）GET /api/tickets/submissions/{id}/（查询状态）POST /api/tickets/{id}/resolve/（完成工单）
流程服务
GET/POST /api/workflows/（流程 CRUD）POST /api/workflows/{id}/publish/（激活流程）
网关聚合
GET /api/overview/（服务数据聚合）GET /api/tickets/queue-metrics/（队列监控）GET /api/healthz（健康检查）

后续规划
认证增强：集成 JWT/OIDC 实现单点登录、多租户隔离
流程扩展：对接 Camunda/Zeebe 支持复杂流程（并行网关、定时任务）
前端优化：引入 React Flow 实现可视化流程拖拽
监控补充：增加 Prometheus + Grafana 监控（网关 / 队列 / 数据库）
贡献指南
Fork 本仓库到个人账号
创建特性分支：git checkout -b feature/your-feature
提交代码：git commit -m "add: 新增XX功能"
推送分支：git push origin feature/your-feature
提交 Pull Request 到主仓库
许可证
本项目基于 MIT License 开源，可自由使用、修改和分发。
