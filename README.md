PFlow v3
** 
 
 
 
PFlow v3 是面向流程编排与工单协同的端到端解决方案，采用 Django 5 微服务 + React 18 前端 + Go API Gateway 架构，支持服务独立部署扩展，通过网关聚合能力，前端统一接入 BFF 接口。
目录
架构概览
本地环境准备
数据库初始化
环境变量配置
依赖安装
服务启动
测试验证
技术选型评估
API 约定
后续规划
贡献指南
许可证
架构概览
目录结构
PFlow-v3/
├── apps/
│   └── frontend/          # React + Vite 管理控制台（可视化操作）
├── services/
│   ├── gateway/           # Go API Gateway（chi 框架，聚合4个领域服务）
│   ├── form/              # 表单建模服务（管理表单结构/字段）
│   ├── identity/          # 身份服务（用户/角色权限）
│   ├── ticket/            # 工单服务（工单生命周期管理）
│   └── workflow/          # 流程定义服务（流程模板/步骤）
├── scripts/
│   └── postgres/init.sql  # 数据库初始化脚本（创建库/角色）
├── .env.example           # 环境变量示例
└── README.md              # 项目文档（本文档）

核心技术栈
模块
技术选型
核心作用
API Gateway
Go 1.21、chi v5、net/http
接口聚合、反向代理、高并发
领域服务
Django 5、DRF、Celery 5
业务逻辑、异步任务
数据存储
PostgreSQL 16（服务独立库）
结构化数据持久化
消息队列
Redis 7 + Celery Worker
工单峰值流量处理
前端
React 18、Vite、Chakra UI、TanStack Query
可视化控制台
依赖管理
pip、npm、go mod
多语言依赖管理

本地环境准备
需提前安装以下依赖（不维护 Docker Compose，容器化可自行编排）：
Go 1.21+（Gateway 运行）
Python 3.12+（Django 服务）
Node.js 18+ + npm（前端依赖）
PostgreSQL 16（建议企业自建 / 包管理器安装）
Redis 7+（开启持久化，用于 Celery）
可选工具：direnv/dotenv（环境变量）、pgcli（PostgreSQL 客户端）
数据库初始化
每个服务需独立数据库与角色，通过脚本一键创建（幂等执行）：
（可选）编辑脚本调整密码 / 库名：
vim scripts/postgres/init.sql

执行脚本（需启动 PostgreSQL，替换 postgres 为管理员账户）：
psql -U postgres -h 127.0.0.1 -f scripts/postgres/init.sql

脚本作用：
创建 4 个数据库：pflow_form、pflow_identity、pflow_ticket、pflow_workflow
为每个库创建同名角色（如 pflow_form 角色拥有对应库权限）
环境变量配置
复制示例配置文件：
cp .env.example .env

关键变量说明（未设置则用默认值）：
服务
核心变量（示例值）
Go Gateway
GATEWAY_PORT=8000、FORM_SERVICE_URL=http://localhost:8001
Form 服务
FORM_DATABASE_URL=postgresql://pflow_form:pflow_form@localhost:5432/pflow_form
Identity 服务
IDENTITY_DATABASE_URL=postgresql://pflow_identity:pflow_identity@localhost:5432/pflow_identity
Ticket 服务
TICKET_BROKER_URL=redis://localhost:6379/0、TICKET_QUEUE_NAME=ticket_submissions
Workflow 服务
WORKFLOW_DATABASE_URL=postgresql://pflow_workflow:pflow_workflow@localhost:5432/pflow_workflow
前端
VITE_GATEWAY_URL=http://localhost:8000/api/

依赖安装
1. Go Gateway 依赖
cd services/gateway
go mod tidy  # 自动安装 chi 等依赖

2. Django 领域服务依赖（4 个服务操作相同）
以 Form 服务为例，其他服务（identity/ticket/workflow）重复此步骤：
# 进入服务目录
cd services/form
# 创建并激活虚拟环境
python3 -m venv .venv
source .venv/bin/activate  # Windows：.venv\Scripts\activate
# 安装依赖
pip install --upgrade pip
pip install -r requirements.txt

3. 前端依赖
cd apps/frontend
npm install  # 安装 React、Vite 等依赖

服务启动
建议打开多个终端，按以下顺序启动（Gateway 依赖其他服务地址）：
1. 启动 Redis（先启动，用于 Celery 队列）
# macOS（brew）
brew services start redis
# Linux（systemd）
sudo systemctl start redis
# 验证：返回 PONG 即正常
redis-cli ping

2. 启动 Celery Worker（Ticket 服务目录）
cd services/ticket
source .venv/bin/activate
# 监听工单队列，输出 info 级日志
celery -A ticket_service worker --loglevel=info

⚠️ 保持终端运行，关闭则无法处理工单任务
3. 启动 Django 领域服务
Form 服务（端口 8001）
cd services/form
source .venv/bin/activate
export FORM_DATABASE_URL=postgresql://pflow_form:pflow_form@localhost:5432/pflow_form
python manage.py runserver 0.0.0.0:8001

Identity 服务（端口 8002）
cd services/identity
source .venv/bin/activate
export IDENTITY_DATABASE_URL=postgresql://pflow_identity:pflow_identity@localhost:5432/pflow_identity
python manage.py runserver 0.0.0.0:8002

Ticket 服务（端口 8003）
cd services/ticket
source .venv/bin/activate
export TICKET_DATABASE_URL=postgresql://pflow_ticket:pflow_ticket@localhost:5432/pflow_ticket
export TICKET_BROKER_URL=redis://localhost:6379/0
python manage.py runserver 0.0.0.0:8003

Workflow 服务（端口 8004）
cd services/workflow
source .venv/bin/activate
export WORKFLOW_DATABASE_URL=postgresql://pflow_workflow:pflow_workflow@localhost:5432/pflow_workflow
python manage.py runserver 0.0.0.0:8004

4. 启动 Go API Gateway（端口 8000）
cd services/gateway
# 导出环境变量（或通过 .env 加载）
export GATEWAY_PORT=8000
export FORM_SERVICE_URL=http://localhost:8001
export IDENTITY_SERVICE_URL=http://localhost:8002
export TICKET_SERVICE_URL=http://localhost:8003
export WORKFLOW_SERVICE_URL=http://localhost:8004
# 启动网关
go run ./cmd/gateway
# 验证：访问 http://localhost:8000/healthz，返回 OK 即正常

5. 启动前端控制台（端口 5173）
cd apps/frontend
# 可选：修改网关地址
# export VITE_GATEWAY_URL=http://your-gateway-url/api/
npm run dev
# 访问 http://localhost:5173 进入控制台

测试验证
1. Django 服务接口测试
# 进入目标服务目录（如 form）
cd services/form
source .venv/bin/activate
# 执行单元测试（验证接口/模型逻辑）
python manage.py test

2. 前端构建验证
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
