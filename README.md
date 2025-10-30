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

- 后端：Django 5 · Django REST Framework · django-cors-headers · requests
- 数据库：PostgreSQL 16（每个服务拥有独立库）
- 前端：React 18 · Vite · Chakra UI · TanStack Query · Axios

## 本地环境准备

1. Python 3.12+
2. Node.js 18+ 与 npm
3. PostgreSQL 16（建议通过包管理器或企业自建数据库，而非 Docker Compose）
4. 可选：Redis/Kafka/Camunda 等后续拓展依赖

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

也可以通过 `psql`/`pgcli` 或企业内部数据库平台完成建库操作。

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
| Ticket | `TICKET_DATABASE_URL` |
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
python manage.py runserver 0.0.0.0:8003

# Workflow service
cd services/workflow
source .venv/bin/activate
export WORKFLOW_DATABASE_URL=postgresql://pflow_workflow:pflow_workflow@localhost:5432/pflow_workflow
python manage.py runserver 0.0.0.0:8004
```

所有服务启动后，API 网关会在 `/api` 下转发 CRUD 接口，同时提供 `/api/overview/` 聚合指标与 `/api/healthz/` 健康探针。

## 前端控制台

```bash
cd apps/frontend
npm install
npm run dev
```

默认情况下前端通过 `VITE_GATEWAY_URL` 指向 `http://localhost:8000/api/`，可在 `.env` 或命令行中修改。运行后在 `http://localhost:5173` 访问控制台，可体验表单库、工单面板、流程设计器与系统概览模块。

## 测试

各 Django 服务均内置基础的 API 测试，可在对应目录运行：

```bash
python manage.py test
```

前端可执行：

```bash
npm run build
```

## API 约定

- 表单服务：`/api/forms/` 提供 CRUD，Schema 与字段以 JSON 表达。
- 身份服务：`/api/users/` 维护协作者列表。
- 工单服务：`/api/tickets/` 支持状态筛选、`POST /api/tickets/{id}/resolve/` 快捷完成工单。
- 流程服务：`/api/workflows/` 管理流程与步骤，`POST /api/workflows/{id}/publish/` 激活流程。
- 网关：统一转发上述接口，`GET /api/overview/` 聚合各服务的数据量与状态分布。

如需对接其他系统，可直接消费各服务 API，或在 Gateway 中新增聚合路由。

## 后续规划

1. 集成认证与多租户能力（JWT/OIDC）。
2. 引入 Celery/Async worker 处理长耗时任务。
3. 扩展流程服务与 Camunda/Zeebe 的互操作。
4. 在前端引入可视化拖拽编排（如 React Flow）。

欢迎在此基础上继续扩展业务能力，或根据企业需求自定义部署拓扑。
