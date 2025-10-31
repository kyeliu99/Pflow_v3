# PFlow v3

PFlow v3 提供了一个面向流程编排与工单协同的端到端解决方案。本仓库以 **Django 5 微服务 + React 18 前端控制台** 的方式实现，每个领域服务都可以独立部署与扩展，API Gateway 负责聚合对外能力，前端通过统一的 BFF 接口访问后端能力。

## 架构概览

```
apps/
  frontend/          # React + Vite 管理控制台
services/
  gateway/           # Go API Gateway（chi），聚合四个领域服务
  form/              # 表单建模服务，管理表单及字段
  identity/          # 身份服务，维护协作者账户与角色
  ticket/            # 工单服务，负责工单生命周期
  workflow/          # 流程定义服务，维护流程及步骤
```

技术栈全部基于官方与主流开源生态：

- API Gateway：Go 1.21 · chi v5 · 标准库 `net/http`
- 领域服务：Django 5 · Django REST Framework · Celery 5 · django-cors-headers · requests
- 数据库：PostgreSQL 16（每个服务拥有独立库）
- 消息队列：Redis 7 + Celery Worker（处理工单提交峰值）
- 前端：React 18 · Vite · Chakra UI · TanStack Query · Axios

## 本地环境准备

1. Go 1.21+
2. Python 3.12+
3. Node.js 18+ 与 npm
4. PostgreSQL 16（建议通过包管理器或企业自建数据库，而非 Docker Compose）
5. Redis 7+（Celery 队列使用，确保开启持久化或连接高可用实例）
6. 可选：Kafka、Camunda 等后续拓展依赖

> 本仓库不再维护 Docker Compose 方案，如需要容器化可在后续阶段自行编排。

## 数据库初始化

为每个服务创建独立的数据库与角色（可按需调整端口/密码）。仓库提供了脚本 `scripts/postgres/init.sql`，可在安装完 PostgreSQL 后执行：

```bash
psql -U postgres -h 127.0.0.1 -f scripts/postgres/init.sql
```

脚本会幂等地创建以下数据库：`pflow_form`、`pflow_identity`、`pflow_ticket`、`pflow_workflow`，并为每个数据库配置同名角色。若需调整密码或数据库名，可在执行前编辑脚本或在企业内部数据库平台上执行等价 SQL。

## 环境变量

复制根目录的示例配置后按需修改（例如数据库密码、端口）：

```bash
cp .env.example .env
```

`.env` 中的变量仅作为参考，实际运行时可以在 shell 中 `export` 或使用 `direnv`/`dotenv` 管理。各服务的 Django 配置会优先读取以下变量：

| 服务 | 关键变量 |
| --- | --- |
| Gateway (Go) | `GATEWAY_PORT`、`GATEWAY_REQUEST_TIMEOUT`、`GATEWAY_SHUTDOWN_GRACE`、`FORM_SERVICE_URL`、`IDENTITY_SERVICE_URL`、`TICKET_SERVICE_URL`、`WORKFLOW_SERVICE_URL` |
| Form | `FORM_DATABASE_URL`、`DJANGO_ALLOWED_HOSTS`、`DJANGO_SECRET_KEY` |
| Identity | `IDENTITY_DATABASE_URL` |
| Ticket | `TICKET_DATABASE_URL`、`TICKET_BROKER_URL`、`TICKET_RESULT_BACKEND`、`TICKET_QUEUE_NAME` |
| Workflow | `WORKFLOW_DATABASE_URL` |
| Frontend | `VITE_GATEWAY_URL` |

未显式设置时会回退到 `.env.example` 中的默认值（本地运行 localhost + 800x 端口）。

## 安装依赖

建议为每个服务创建独立虚拟环境。以下命令以 `venv` 为例：

```bash
# Form service
direnv allow .  # 如使用 direnv，可在每个服务目录配置 .envrc
python3 -m venv .venv
source .venv/bin/activate
pip install --upgrade pip
pip install -r requirements.txt
```

对 `services/identity`、`services/ticket`、`services/workflow`、`services/form` 重复上述步骤。前端在 `apps/frontend` 中执行 `npm install`。Gateway 由于采用 Go，可直接在 `services/gateway` 执行 `go mod tidy` 安装依赖。

## 运行数据库迁移

每个 Django 服务都包含手工维护的初始迁移，首次启动前执行：

```bash
cd services/form
source .venv/bin/activate
python manage.py migrate

cd ../identity
source .venv/bin/activate
python manage.py migrate

cd ../ticket
source .venv/bin/activate
python manage.py migrate

cd ../workflow
source .venv/bin/activate
python manage.py migrate
```

## 启动 Redis 与 Celery Worker

工单服务的高并发写入通过 Celery + Redis 队列处理，请确保 Redis 已运行：

```bash
# 以 macOS 为例
brew install redis
brew services start redis

# 或使用 Linux 原生包管理器
sudo systemctl start redis
```

随后在 `services/ticket` 目录启动 Celery worker：

```bash
cd services/ticket
source .venv/bin/activate
celery -A ticket_service worker --loglevel=info
```

Celery 会监听 `TICKET_QUEUE_NAME` 队列（默认 `ticket_submissions`），并自动处理通过 API 提交的工单创建任务。

## 启动服务

为方便调试，推荐在多个终端窗口中分别启动服务：

```bash
# API Gateway (Go)
cd services/gateway
export FORM_SERVICE_URL=http://localhost:8001
export IDENTITY_SERVICE_URL=http://localhost:8002
export TICKET_SERVICE_URL=http://localhost:8003
export WORKFLOW_SERVICE_URL=http://localhost:8004
export GATEWAY_PORT=8000
go run ./cmd/gateway

# Form service
cd services/form
source .venv/bin/activate
export FORM_DATABASE_URL=postgresql://pflow_form:pflow_form@localhost:5432/pflow_form
python manage.py runserver 0.0.0.0:8001

# Identity service
cd services/identity
source .venv/bin/activate
export IDENTITY_DATABASE_URL=postgresql://pflow_identity:pflow_identity@localhost:5432/pflow_identity
python manage.py runserver 0.0.0.0:8002

# Ticket service
cd services/ticket
source .venv/bin/activate
export TICKET_DATABASE_URL=postgresql://pflow_ticket:pflow_ticket@localhost:5432/pflow_ticket
export TICKET_BROKER_URL=redis://localhost:6379/0
export TICKET_RESULT_BACKEND=redis://localhost:6379/0
python manage.py runserver 0.0.0.0:8003

# Workflow service
cd services/workflow
source .venv/bin/activate
export WORKFLOW_DATABASE_URL=postgresql://pflow_workflow:pflow_workflow@localhost:5432/pflow_workflow
python manage.py runserver 0.0.0.0:8004
```

> 提示：工单队列需要独立的 Celery worker（见上文），请在另一个终端保持 `celery -A ticket_service worker` 运行，以避免高并发场景下的请求丢失。

所有服务启动后，Go API 网关会在 `/api` 下转发 CRUD 接口，同时提供 `/api/overview/` 聚合指标与 `/healthz` 健康探针。

## 前端控制台

```bash
cd apps/frontend
npm install
npm run dev
```

默认情况下前端通过 `VITE_GATEWAY_URL` 指向 `http://localhost:8000/api/`，可在 `.env` 或命令行中修改。运行后在 `http://localhost:5173` 访问控制台，可体验表单库、工单面板、流程设计器与系统概览模块。

> 控制台在提交工单时会自动生成客户端请求 ID，并在队列处理完成后自动刷新列表，避免网络抖动导致的重复提交。

## 测试

各 Django 服务均内置基础的 API 测试，可在对应目录运行：

```bash
python manage.py test
```

前端可执行：

```bash
npm run build
```

## 技术选型评估：Go Gateway + Django 领域服务

在最新版本中，API Gateway 已替换为 Go 语言实现，以获得更轻量的部署方式与更高的并发处理能力；而四个领域服务仍保留 Django 生态，原因如下：

- **Gateway 的优势**：Go 的并发模型和单二进制部署便于在交付环境中横向扩展，同时借助 chi 等主流路由器即可快速实现反向代理与聚合逻辑，减少 Python 运行时的额外开销。
- **领域服务建模效率**：Django + DRF 依然在表单、工单、流程等数据密集型业务中具备极高的开发效率，ORM、Admin、权限体系和 Celery 集成都无需重复造轮子。
- **官方生态与长期维护**：Go 网关仅依赖标准库与 chi（Go 社区主流项目），Django 侧依赖 Django、DRF、Celery、Redis 驱动等官方或基金会维护的库，彻底避免个人仓库依赖。
- **异步与负载能力**：Celery + Redis 的提交队列继续承担工单高峰流量，前端防重复提交逻辑配合网关快速响应，实现端到端的“防呆”体验。

因此，本方案以 Go 负责网关层的高并发代理，Django 负责业务建模与异步任务，兼顾性能、迭代效率与团队技能结构。

## API 约定

- 表单服务：`/api/forms/` 提供 CRUD，Schema 与字段以 JSON 表达。
- 身份服务：`/api/users/` 维护协作者列表。
- 工单服务：`POST /api/tickets/submissions/` 通过队列异步创建工单、`GET /api/tickets/submissions/{id}/` 查询排队结果、`POST /api/tickets/{id}/resolve/` 快捷完成工单。
- 流程服务：`/api/workflows/` 管理流程与步骤，`POST /api/workflows/{id}/publish/` 激活流程。
- 网关：统一转发上述接口，`GET /api/overview/` 聚合各服务的数据量与状态分布，并提供 `GET /api/tickets/queue-metrics/` 反馈队列运行状况。

如需对接其他系统，可直接消费各服务 API，或在 Gateway 中新增聚合路由。

## 后续规划

1. 集成认证与多租户能力（JWT/OIDC）。
2. 扩展流程服务与 Camunda/Zeebe 的互操作。
3. 在前端引入可视化拖拽编排（如 React Flow）。

欢迎在此基础上继续扩展业务能力，或根据企业需求自定义部署拓扑。
