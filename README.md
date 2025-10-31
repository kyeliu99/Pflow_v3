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

### 1. 依赖准备

- Go 1.21+
- Node.js 18+
- PostgreSQL 15+（可部署在本地或远程服务器）
- `psql` 命令行工具（随 PostgreSQL 一并安装）
- （可选）Docker & Docker Compose —— 仅当你希望容器化依赖服务时使用

### 2. 初始化数据库（无需 Docker）

在已有 PostgreSQL 实例的前提下，可直接执行仓库自带的引导脚本创建默认账号与数据库：

```bash
# 以本地 PostgreSQL 默认超级用户为例，提前导出密码（或使用 .pgpass 文件）
export PGPASSWORD=<postgres 密码>

# 可通过以下环境变量覆盖连接信息：
#   POSTGRES_HOST / POSTGRES_PORT          —— PostgreSQL 地址（默认 localhost:5432）
#   POSTGRES_SUPERUSER                    —— 具有创建数据库权限的账号（默认 postgres）
#   POSTGRES_DB                           —— 连接时使用的数据库（默认 postgres）
./scripts/postgres/bootstrap.sh
```

`bootstrap.sh` 会重复执行 `scripts/postgres/init.sql` 并确保幂等：

- 若不存在 `pflow` 角色则自动创建并设置口令 `pflow`
- 若不存在 `pflow` 数据库则创建并将所有权授予 `pflow`

如无法使用脚本，也可以手动执行以下 SQL：

```sql
CREATE ROLE pflow LOGIN PASSWORD 'pflow';
CREATE DATABASE pflow OWNER pflow;
```

完成后，微服务即可使用 `.env` 中的默认 `POSTGRES_DSN=postgres://pflow:pflow@localhost:5432/pflow?sslmode=disable` 进行连接。

### 3. （可选）使用 Docker Compose 启动依赖

若希望在本地快速拉起一套隔离的依赖服务，可继续使用项目根目录的 `docker-compose.yml`（示例见后文）。

- 默认 compose 会一次性拉起 PostgreSQL、Zookeeper、Kafka 与 Camunda：

```bash
docker compose up -d postgres zookeeper kafka camunda
```

- PostgreSQL 容器启动时同样会自动运行 `scripts/postgres/init.sql`，确保创建 `pflow` 数据库与登录角色。
- 如果此前已经启动过旧版本的容器导致卷内缺少该角色，可执行 `docker compose down -v postgres` 清理卷后再启动，或手动进入容器执行  `psql -U postgres -c "CREATE ROLE pflow LOGIN PASSWORD 'pflow';"` 与 `psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE pflow TO pflow;"`。

- PostgreSQL 暴露在 `5432`
- Kafka 暴露在 `9092`（容器互联 `kafka:9092`，宿主机备用监听 `localhost:9092`）
- Camunda/Zeebe 网关暴露在 `26500`（gRPC）与 `8088`（控制台）
- 如果某个容器启动失败，可通过 `docker compose logs <service>` 查看原因

### 4. 配置环境变量

将示例配置复制为仓库根目录的 `.env`（一次即可）：

```bash
cp .env.example .env
```

所有微服务都会自动读取仓库根目录的 `.env`、`.env.local` 以及 `.env.d/*.env` 文件，无需再为每个服务重复拷贝。

> `.env.example` 提供 `FORM_DATABASE_DSN`/`FORM_HTTP_PORT` 等服务级变量：若未配置则自动回退到 `POSTGRES_DSN` 与推荐端口（Gateway=8080、Form=8081、Identity=8082、Ticket=8083、Workflow=8084），也可以继续通过全局 `HTTP_PORT`/`POSTGRES_DSN` 快速覆盖。运行命令前按需设置对应环境变量即可，例如 `FORM_HTTP_PORT=9001 go run ./cmd/main.go`。

如需加载额外的配置文件，可通过 `PFLOW_ENV_FILES` 指定逗号分隔的路径列表。

> `.env` 中的 `POSTGRES_IMAGE`、`ZOOKEEPER_IMAGE`、`KAFKA_IMAGE`、`CAMUNDA_IMAGE` 变量可按需指向企业私有仓库或镜像加速服务，以避免 Docker Hub 拉取受限。

### 5. 启动微服务

建议在独立终端中分别启动各个服务（默认端口见下表，可按需覆盖 `HTTP_PORT`）：

| 服务 | 目录 | 默认端口 | 启动命令 |
| --- | --- | --- | --- |
| API Gateway | `services/gateway` | 8080 | `go run ./cmd/main.go` |
| Form Service | `services/form` | 8081 | `go run ./cmd/main.go` |
| Identity Service | `services/identity` | 8082 | `go run ./cmd/main.go` |
| Ticket Service | `services/ticket` | 8083 | `go run ./cmd/main.go` |
| Workflow Service | `services/workflow` | 8084 | `go run ./cmd/main.go` |

> 服务在启动时会调用 `cfg.DatabaseDSN(<service>)` 与 `cfg.ResolveServiceHTTPPort(<service>, <fallback>)`：只需在 `.env` 或运行命令前设置 `FORM_DATABASE_DSN`、`TICKET_HTTP_PORT` 等变量即可让组件无缝连接不同的数据库实例或监听端口。`libs/shared/database.ConnectWithDSN` 会缓存命名连接，便于在同一进程中复用多个数据源。

启动顺序建议为：先运行依赖基础设施与 API Gateway，再依次启动领域服务。可借助 `air`、`fresh` 等热加载工具提升开发效率。

### 6. 前端控制台

```bash
cd apps/frontend
# 验证代码可正常打包（无语法错误）
npm run build

技术选型评估
采用「Go 网关 + Django 业务服务」混合方案，核心考量：
Go Gateway 优势：
单二进制部署，适合横向扩展
goroutine 并发模型，高并发处理优于 Python
依赖精简（chi + 标准库），维护成本低
Django 领域服务优势：
数据密集型业务效率高：ORM、序列化、权限开箱即用
生态成熟：Celery 异步、Django Admin 调试便捷
团队适配：降低前端 / 运营学习成本
整体平衡：
Go 负责「流量入口」高并发，Django 负责「业务核心」快速建模
依赖均为官方 / 主流开源项目，无个人仓库风险
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
