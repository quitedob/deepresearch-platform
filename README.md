# Go Deep Research

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Vue-3.x-4FC08D?style=flat&logo=vue.js" alt="Vue Version">
  <img src="https://img.shields.io/badge/Gin-Web-green" alt="Gin">
  <img src="https://img.shields.io/badge/Eino-AI-orange" alt="CloudWeGo Eino">
  <img src="https://img.shields.io/badge/GORM-PostgreSQL-blue" alt="GORM PostgreSQL">
  <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg" alt="PRs Welcome">
</p>

基于 **Gin + CloudWeGo Eino** 混合架构的新一代 AI 研究平台，集成多智能体协作、自主研究工作流、智能题目生成和完整的会员管理体系。

## ✨ 核心功能

### 🤖 多模型支持
| 提供商 | 模型 | 特点 |
|--------|------|------|
| **DeepSeek** | deepseek-chat | 非思考模式，快速响应 |
| | deepseek-reasoner | 思考模式，深度推理 |
| **智谱AI** | glm-4.7 | 旗舰模型，深度思考 |
| | glm-4.5-air | 高性价比模型 |
| **GLM Coding** | glm-4.7 / glm-4.5-air | 编程专用API |
| **Ollama** | gemma3:4b / 12b | 本地部署 |
| | qwen3:8b | 通义千问本地版 |
| **OpenRouter** | kimi-k2:free | 免费模型 |

### 💬 智能聊天系统
- **实时对话**: 流式/非流式输出，SSE实时推送
- **会话管理**: 创建/更新/删除/列出聊天会话
- **消息历史**: 完整的对话历史和上下文管理
- **上下文状态**: 实时监控token使用，智能总结
- **深度思考**: 自动切换推理模型进行复杂分析
- **联网搜索**: 集成WebSearchTool获取实时信息

### 🔬 智能研究系统
**多智能体协作架构**:
- **Planner Agent** - 研究规划器
- **Research Agent** - 执行研究者
- **Evaluator Agent** - 结果评估者
- **Critic Agent** - 批评分析者
- **Structured Report Agent** - 结构化报告
- **Parallel Agent** - 并行执行器

**三种研究模式**:
- 快速研究 (quick) - 快速获取信息
- 深度研究 (deep) - 多步骤深入分析
- 综合研究 (comprehensive) - 全方位研究

**实时监控**:
- SSE流式显示研究进度
- 证据链追踪和可靠性分析
- 支持Markdown/JSON导出

### 📄 扩展功能（数据模型支持）
- **AI论文生成**: 数据模型已就绪，支持多种学术格式
- **AI题目生成**: 数据模型已就绪，支持多种题型
- **会员与配额系统**: 完整的数据库模型支持
- **通知系统**: 数据库模型已实现

> 注：部分功能的数据模型已创建，API 端点开发中

## 🏗️ 技术架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Vue 3 Frontend                               │
│              (Vite + Pinia + Vue Router + Element Plus)             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │ ChatContainer│  │ResearchSpace │  │  AISpace     │             │
│  │ AdminPanel   │  │Settings      │  │  Components  │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
└─────────────────────────────────────────────────────────────────────┘
                              ↕ HTTP/SSE
┌─────────────────────────────────────────────────────────────────────┐
│                      Go Backend (Gin Web API)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │ Auth Handler │  │ Chat Handler │  │ Research     │             │
│  │ LLM Handler  │  │ Middleware   │  │   Handler    │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
└─────────────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────────────┐
│                 AI Agent Layer (CloudWeGo Eino)                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              Multi-Agent Orchestration                       │   │
│  │  Planner → Research → Evaluator → Critic → Report           │   │
│  └─────────────────────────────────────────────────────────────┘   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │ LLMScheduler │  │ ChatModel    │  │ Tool Registry│             │
│  │ StreamMgr    │  │ ConfigMgr    │  │ Reliability  │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
└─────────────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────────────┐
│                   Infrastructure Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │ PostgreSQL   │  │ Redis        │  │ LLM APIs     │             │
│  │ (GORM)       │  │ (Cache)      │  │ DeepSeek     │             │
│  │              │  │              │  │ Zhipu/GLM    │             │
│  │              │  │              │  │ Ollama       │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
└─────────────────────────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 环境要求
- Go 1.21+
- Node.js 18+
- PostgreSQL 14+
- Redis 6+

### 1. 克隆项目
```bash
git clone https://github.com/quitedob/deepresearch-platform.git
cd deepresearch-platform
```

### 2. 配置环境变量
```bash
cp .env.example .env
# 编辑 .env 文件，填入你的 API Keys
```

必需的环境变量：
```env
# LLM API Keys
DEEPSEEK_API_KEY=your_deepseek_api_key
ZHIPU_API_KEY=your_zhipu_api_key
OPENROUTER_API_KEY=your_openrouter_api_key  # 可选

# Database
DB_PASSWORD=your_db_password

# JWT Secret
JWT_SECRET=your_jwt_secret

# Admin User (可选)
ADMIN_EMAIL=admin@example.com
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123
```

### 3. 启动后端
```bash
# 安装依赖
go mod download

# 编译
go build -o server.exe ./src/cmd/server

# 运行
./server.exe -config src/configs/config.yaml
```

### 4. 启动前端
```bash
cd vue
npm install
npm run dev
```

访问 http://localhost:5173 开始使用！

## 👤 默认用户

系统启动时会自动创建默认管理员账户：

| 字段 | 默认值 | 环境变量 |
|------|--------|----------|
| 邮箱 | `admin@example.com` | `ADMIN_EMAIL` |
| 用户名 | `admin` | `ADMIN_USERNAME` |
| 密码 | `admin123` | `ADMIN_PASSWORD` |

> ⚠️ **安全提示**: 生产环境请务必通过环境变量修改默认密码！

```env
# .env 文件中配置管理员
ADMIN_EMAIL=your_admin@example.com
ADMIN_USERNAME=your_admin
ADMIN_PASSWORD=your_secure_password
```

## 📁 项目结构

```
go-deep-research/
├── src/
│   ├── cmd/server/          # 应用入口
│   ├── configs/             # 配置文件 (config.yaml, models.yaml)
│   ├── internal/
│   │   ├── handler/         # HTTP 处理器
│   │   ├── middleware/      # Gin 中间件
│   │   ├── repository/      # 数据访问层 (DAO + Model)
│   │   │   ├── dao/         # 数据访问对象
│   │   │   └── model/       # GORM 数据模型
│   │   ├── service/         # 业务逻辑层
│   │   ├── pkg/             # 工具包
│   │   │   ├── auth/        # JWT + 密码加密
│   │   │   ├── llm/         # LLM 提供商实现
│   │   │   ├── eino/        # CloudWeGo Eino 封装
│   │   │   │   ├── agent/   # AI Agents (Planner, Research, Evaluator, Critic)
│   │   │   │   ├── model/   # Eino 模型适配器
│   │   │   │   └── tool/    # MCP 工具实现 (WebSearch, Arxiv, Wikipedia, ZRead)
│   │   │   ├── paper/       # 论文生成工具
│   │   │   └── tools/       # 工具接口
│   │   ├── cache/           # Redis 缓存
│   │   ├── database/        # 数据库初始化和迁移
│   │   ├── monitoring/      # Prometheus 监控
│   │   └── types/           # 请求/响应类型
│   └── infrastructure/      # 基础设施层
├── vue/                     # Vue 3 前端
│   ├── src/
│   │   ├── api/             # API 调用
│   │   ├── components/      # Vue 组件
│   │   ├── views/           # 页面视图
│   │   ├── stores/          # Pinia 状态管理
│   │   └── router/          # Vue Router 配置
│   └── ...
└── docs/                    # 文档
```

## 📡 API 端点

### 🔐 认证 (`/api/v1/auth`)
| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/register` | 用户注册 |
| POST | `/login` | 用户登录 |
| POST | `/refresh` | 刷新Token |

### 💬 聊天功能 (`/api/v1/chat`)
| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/sessions` | 创建聊天会话 |
| GET | `/sessions` | 获取会话列表 |
| GET | `/sessions/:session_id` | 获取会话详情 |
| DELETE | `/sessions/:session_id` | 删除会话 |
| POST | `/sessions/:session_id/messages` | 发送消息 |
| GET | `/sessions/:session_id/messages` | 获取消息列表 |
| GET | `/sessions/:session_id/stream` | 流式消息 |
| PUT | `/sessions/:session_id/provider` | 更新提供商 |

### 🔬 深度研究 (`/api/v1/research`)
| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/sessions` | 启动研究会话 |
| GET | `/sessions` | 获取研究会话列表 |
| GET | `/sessions/:session_id` | 获取研究会话详情 |
| GET | `/sessions/:session_id/results` | 获取研究结果 |
| GET | `/sessions/:session_id/tasks` | 获取研究任务 |
| GET | `/sessions/:session_id/stream` | 流式获取研究进度(SSE) |
| POST | `/sessions/:session_id/cancel` | 取消研究 |

### 🤖 LLM管理 (`/api/v1/llm`)
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/providers` | 获取LLM提供商列表 |
| GET | `/providers/:provider/metrics` | 获取提供商指标 |
| GET | `/models` | 获取所有可用模型 |
| POST | `/test` | 测试提供商连接 |

## 🎯 核心交互功能

| 功能 | 后端实现 | 说明 |
|------|----------|------|
| **聊天对话** | `ChatService` + `LLMScheduler` | 支持多模型、流式/非流式输出 |
| **深度研究** | `ResearchService` + 多 Agent 系统 | 规划→执行→评估→报告全流程 |
| **模型切换** | `LLMScheduler` | 动态切换 LLM 提供商和模型 |

### 研究流程详解
```
┌─────────────────────────────────────────────────────────────┐
│  1. Planner Agent    - 分析问题，生成研究计划               │
│  2. Research Agent   - 使用工具收集信息                     │
│  3. Evaluator Agent  - 评估信息质量和相关性                 │
│  4. Critic Agent     - 批评分析，找出遗漏                  │
│  5. Structured Report Agent - 生成结构化报告                │
└─────────────────────────────────────────────────────────────┘
```

## 🌟 项目亮点

1. **混合架构** - Gin 处理 Web API + CloudWeGo Eino 处理 AI Agent 编排
2. **多 LLM 提供商** - DeepSeek、智谱AI、Ollama、OpenRouter、GLM Coding Plan
3. **多智能体协作** - 基于 Eino 框架的 ReAct Agent 系统（Planner、Research、Evaluator、Critic、Report）
4. **MCP 工具集成** - WebSearch、Arxiv、Wikipedia、ZRead、WebReader、WebSearchPrime
5. **实时流式输出** - SSE 推送，支持长文本流式展示
6. **完整的数据库迁移** - 自动检查表结构、创建索引、初始化默认数据
7. **Prometheus 监控** - 内置指标收集和监控端点
8. **Redis 缓存支持** - 会话管理和性能优化
9. **灵活的配置系统** - 支持环境变量覆盖和 YAML 配置

## 🚀 快速测试

```bash
# 使用 curl 测试 API

# 1. 登录获取 token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 2. 创建聊天会话 (GLM-4.7)
curl -X POST http://localhost:8080/api/v1/chat/sessions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"title":"测试","provider":"zhipu","model":"glm-4.7"}'

# 3. 发送消息
curl -X POST http://localhost:8080/api/v1/chat/sessions/<SESSION_ID>/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"message":"你好"}'

# 4. 启动深度研究
curl -X POST http://localhost:8080/api/v1/research/sessions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"query":"Go语言的最新发展","mode":"deep"}'

# 5. 获取 LLM 提供商列表
curl -X GET http://localhost:8080/api/v1/llm/providers \
  -H "Authorization: Bearer <TOKEN>"
```

## 🔧 GLM Coding Plan 配置

在 `config.yaml` 中配置 GLM Coding Plan：

```yaml
llm:
  providers:
    zhipu:
      api_key: "your_api_key"
      base_url: https://open.bigmodel.cn/api/paas/v4  # 标准端点
      models:
        - glm-4.7
        - glm-4.5-air

    openai:  # 用于 GLM Coding Plan
      api_key: "your_api_key"
      base_url: https://api.z.ai/api/coding/paas/v4  # Coding专用端点
      models:
        - GLM-5
        - GLM-4.7
        - GLM-4.5-air
```

### Cursor 集成 GLM Coding Plan

```
Provider: OpenAI Compatible
API Key: [你的 GLM API Key]
Base URL: https://api.z.ai/api/coding/paas/v4
Model: GLM-5 (大写)
```

### 模型配置文件 (models.yaml)

项目使用独立的 `models.yaml` 文件管理模型元数据：

- **providers**: 提供商配置（显示名称、启用状态、排序）
- **models**: 模型详细信息（显示名称、描述、上下文长度、能力标签）
- **deep_thinking_models**: 深度思考模型映射关系

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 贡献流程
1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

### 开发规范
- 遵循 Go 语言代码规范
- 添加必要的单元测试
- 更新相关文档
- 提交信息清晰明确

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 👤 作者

**quitedob** - [GitHub](https://github.com/quitedob)
- 📧 Email: dobqop999@gmail.com

## 🙏 致谢

- [CloudWeGo Eino](https://github.com/cloudwego/eino) - 优秀的 Go 语言 AI Agent 应用框架
- [DeepSeek](https://www.deepseek.com/) - 提供强大的推理模型
- [智谱AI](https://www.bigmodel.cn/) - 提供 GLM 系列模型
- [Gin](https://github.com/gin-gonic/gin) - 高性能 Go Web 框架
- [GORM](https://gorm.io/) - Go ORM 库
- 所有贡献者和使用者

## 🔗 相关链接

- [Eino 官方文档](https://www.cloudwego.io/zh/docs/eino/)
- [Eino GitHub](https://github.com/cloudwego/eino)
- [GLM Coding Plan](https://www.bigmodel.cn/glm-coding)
- [DeepSeek API](https://platform.deepseek.com/)
- [Ollama](https://ollama.com/) - 本地模型部署

---

<div align="center">

**如果这个项目对你有帮助，请给一个 ⭐️ Star！**

Made with ❤️ by [quitedob](https://github.com/quitedob)

</div>
