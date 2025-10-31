# PFlow v3

PFlow v3 提供了一个面向流程编排与工单协同的端到端解决方案。本仓库以 **Django 5 微服务 + React 18 前端控制台** 的方式实现，每个领域服务都可以独立部署与扩展，API Gateway 负责聚合对外能力，前端通过统一的 BFF 接口访问后端能力。

## 架构概览

```
apps/
  frontend/          # React + Vite 管理控制台
services/
  gateway/           # Django API Gateway，聚合四个领域服务
  form/              # 表单建模服务，管理表单及字段
  identity/          # 身份服务，维护协作者账户与角色
  ticket/            # 工单服务，负责工单生命周期
  workflow/          # 流程定义服务，维护流程及步骤
```

技术栈全部基于官方与主流开源生态：

- 后端：Django 5 · Django REST Framework · Celery 5 · django-cors-headers · requests
- 数据库：PostgreSQL 16（每个服务拥有独立库）
- 消息队列：Redis 7 + Celery Worker（处理工单提交峰值）
- 前端：React 18 · Vite · Chakra UI · TanStack Query · Axios

## 本地环境准备

1. Python 3.12+
2. Node.js 18+ 与 npm
3. PostgreSQL 16（建议通过包管理器或企业自建数据库，而非 Docker Compose）
4. Redis 7+（Celery 队列使用，确保开启持久化或连接高可用实例）
5. 可选：Kafka、Camunda 等后续拓展依赖

> 本仓库不再维护 Docker Compose 方案，如需要容器化可在后续阶段自行编排。

## 数据库初始化

为每个服务创建独立的数据库与角色（可按需调整端口/密码）：

```sql
CREATE ROLE pflow_gateway LOGIN PASSWORD 'pflow_gateway';
CREATE ROLE pflow_form LOGIN PASSWORD 'pflow_form';
CREATE ROLE pflow_identity LOGIN PASSWORD 'pflow_identity';
CREATE ROLE pflow_ticket LOGIN PASSWORD 'pflow_ticket';
CREATE ROLE pflow_workflow LOGIN PASSWORD 'pflow_workflow';

CREATE DATABASE pflow_gateway OWNER pflow_gateway;
CREATE DATABASE pflow_form OWNER pflow_form;
CREATE DATABASE pflow_identity OWNER pflow_identity;
CREATE DATABASE pflow_ticket OWNER pflow_ticket;
CREATE DATABASE pflow_workflow OWNER pflow_workflow;
```

<<<<<<< HEAD
也可以通过 `psql`/`pgcli` 或企业内部数据库平台完成建库操作。
=======
- 首次启动 PostgreSQL 会自动执行 `scripts/postgres/init.sql`，确保创建 `pflow` 数据库与登录角色。
- 如果此前已经启动过旧版本的容器导致卷内缺少该角色，可执行 `docker compose down -v postgres` 清理卷后再启动，或手动进入容器执行 `psql -U postgres -c "CREATE ROLE pflow LOGIN PASSWORD 'pflow';"` 与 `psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE pflow TO pflow;"`。

- PostgreSQL 暴露在 `5432`
- Kafka 暴露在 `9092`（容器互联 `kafka:9092`，宿主机备用监听 `localhost:9092`）
- Camunda/Zeebe 网关暴露在 `26500`（gRPC）与 `8088`（控制台）
- 如果某个容器启动失败，可通过 `docker compose logs <service>` 查看原因
>>>>>>> main

## 环境变量

复制根目录的示例配置后按需修改（例如数据库密码、端口）：

```bash
cp .env.example .env
```

`.env` 中的变量仅作为参考，实际运行时可以在 shell 中 `export` 或使用 `direnv`/`dotenv` 管理。各服务的 Django 配置会优先读取以下变量：

| 服务 | 关键变量 |
| --- | --- |
| Gateway | `GATEWAY_DATABASE_URL`、`FORM_SERVICE_URL`、`IDENTITY_SERVICE_URL`、`TICKET_SERVICE_URL`、`WORKFLOW_SERVICE_URL` |
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

对 `services/identity`、`services/ticket`、`services/workflow`、`services/gateway` 重复上述步骤。前端在 `apps/frontend` 中执行 `npm install`。

## 运行数据库迁移

每个服务都包含手工维护的初始迁移，首次启动前执行：

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

cd ../gateway
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
# API Gateway
cd services/gateway
source .venv/bin/activate
export GATEWAY_DATABASE_URL=postgresql://pflow_gateway:pflow_gateway@localhost:5432/pflow_gateway
export FORM_SERVICE_URL=http://localhost:8001
export IDENTITY_SERVICE_URL=http://localhost:8002
export TICKET_SERVICE_URL=http://localhost:8003
export WORKFLOW_SERVICE_URL=http://localhost:8004
python manage.py runserver 0.0.0.0:8000

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

所有服务启动后，API 网关会在 `/api` 下转发 CRUD 接口，同时提供 `/api/overview/` 聚合指标与 `/api/healthz/` 健康探针。

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

## 技术选型评估：Django vs Go

根据当前需求，微服务需要快速迭代、具备成熟的 ORM/序列化能力，并能够与 Celery 等异步组件无缝集成：

- **快速建模能力**：Django + Django REST Framework 提供开箱即用的模型迁移、序列化与验证体系，适合频繁调整业务字段的场景；Go 则需要手动组合 Gin/Fiber + GORM/SQLC 等组件，开发效率略低。
- **官方生态与长期维护**：本方案完全依赖官方维护或基金会托管的库（Django、DRF、Celery、Redis 驱动等），避免引入个人仓库依赖；Go 虽性能优异，但在表单/流程这类数据密集场景并无决定性优势。
- **异步与水平扩展**：Celery 与 Redis 深度集成，提供任务重试、监控等能力，满足工单高并发创建需求；若后续需要进一步扩容，可通过增加 worker 实例扩展吞吐。
- **多语言协同**：前端和运营团队更熟悉 Django 模型定义与 Admin 后台，可降低学习成本；若需要引入高性能计算模块，可在后续以 gRPC/HTTP 方式接入 Go 服务。

综合评估后，继续采用 Django 生态能够在交付周期、团队技能与稳定性之间取得最佳平衡，同时保留未来按需引入 Go 微服务的空间。

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
