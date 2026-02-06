# Go Deep Research

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Vue-3.x-4FC08D?style=flat&logo=vue.js" alt="Vue Version">
  <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg" alt="PRs Welcome">
</p>

基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) 框架构建的高性能 AI 研究平台，支持多 LLM 提供商、自主研究工作流和实时流式输出。

## ✨ 核心功能

### 🤖 多模型支持
- **DeepSeek**: deepseek-chat（对话）、deepseek-reasoner（深度思考）
- **智谱AI**: glm-4.7（旗舰）、glm-4.5-air（高性价比）
- **Ollama**: gemma3、qwen3 等本地模型

### 💬 智能聊天
- **普通对话**: 流式/非流式输出
- **深度思考**: 自动切换推理模型，支持复杂问题分析
- **联网搜索**: 调用 WebSearchTool 获取实时网络信息
- **聊天记忆**: 上下文累加，支持 128K token 限制管理

### 🔬 深度研究
基于 Eino ReAct Agent 实现的自主研究系统：
- **三大工具**: WebSearch（网络搜索）、ArXiv（学术论文）、Wikipedia（百科知识）
- **多步推理**: planning → executing → synthesis → completed
- **实时进度**: SSE 流式推送研究进度

### 👥 会员体系
- **普通用户**: 10次聊天 + 1次深度研究
- **高级会员**: 50次聊天 + 10次深度研究
- **激活码系统**: 支持批量生成和追踪

### 🛠️ 管理后台
- 用户管理、模型配置、配额管理
- 激活码管理、通知系统
- 聊天记录查询和导出

## 🏗️ 技术架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Vue 3 Frontend                         │
│              (Vite + Pinia + Vue Router)                    │
└─────────────────────────────────────────────────────────────┘
                            ↕ HTTP/SSE
┌─────────────────────────────────────────────────────────────┐
│                    Go Backend (Gin)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Auth API     │  │ Chat API     │  │ Research API │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────┘
                            ↕
┌─────────────────────────────────────────────────────────────┐
│                  Eino Component Layer                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ LLMScheduler │  │ ResearchAgent│  │ Tool Registry│     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────┘
                            ↕
┌─────────────────────────────────────────────────────────────┐
│              Infrastructure Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ PostgreSQL   │  │ Redis        │  │ LLM APIs     │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 环境要求
- Go 1.21+
- Node.js 18+
- PostgreSQL 14+
- Redis 6+

### 1. 克隆项目
```bash
git clone https://github.com/quitedob/go-deep-research.git
cd go-deep-research
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

# Database
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_password
POSTGRES_DB=go_deep_research

# JWT Secret
JWT_SECRET=your_jwt_secret

# System Prompt (可选)
SYSTEM_SAFETY_PROMPT=你是一个有帮助的AI助手...
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
│   ├── configs/             # 配置文件
│   ├── docs/                # 详细文档
│   └── internal/
│       ├── api/             # HTTP API 层
│       │   └── v1/          # API v1 版本
│       ├── eino/            # Eino 组件
│       │   ├── agent/       # 研究 Agent
│       │   ├── model/       # 模型封装
│       │   └── tool/        # 工具实现
│       ├── middleware/      # 中间件
│       ├── repository/      # 数据访问层
│       ├── service/         # 业务逻辑层
│       └── types/           # 类型定义
├── vue/                     # Vue 3 前端
│   ├── src/
│   │   ├── api/             # API 调用
│   │   ├── components/      # 组件
│   │   ├── store/           # Pinia 状态
│   │   └── views/           # 页面
│   └── ...
├── test/                    # 测试脚本
└── ...
```

## 🔧 API 端点

### 认证
- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/refresh` - 刷新 Token

### 聊天
- `POST /api/v1/chat/sessions` - 创建会话
- `GET /api/v1/chat/sessions` - 获取会话列表
- `POST /api/v1/chat/chat` - 发送消息
- `POST /api/v1/chat/chat/stream` - 流式聊天
- `POST /api/v1/chat/chat/web-search` - 联网搜索聊天

### 深度研究
- `POST /api/v1/research/start` - 启动研究
- `GET /api/v1/research/status/:id` - 获取研究状态
- `GET /api/v1/research/stream/:id` - 流式获取进度

### 会员
- `GET /api/v1/membership` - 获取会员信息
- `POST /api/v1/membership/activate` - 激活码激活

## 🎯 三大按钮功能

| 按钮 | 功能 | 后端实现 |
|------|------|----------|
| 🧠 深度思考 | 切换到推理模型 | `getDeepThinkingModel()` 自动切换 |
| 🌐 联网搜索 | 调用 WebSearchTool | `ChatWebSearch()` + 智谱AI web_search |
| 🔬 深度研究 | ReAct Agent 多工具研究 | `ResearchAgent` + ArXiv/Wikipedia/WebSearch |

## 📚 详细文档

- [API 文档](src/docs/API_DOCUMENTATION.md)
- [架构设计](src/docs/ARCHITECTURE.md)
- [开发指南](src/docs/DEVELOPER_SETUP.md)
- [配置参考](src/docs/CONFIGURATION_REFERENCE.md)
- [部署指南](src/docs/DEPLOYMENT_SETUP.md)
- [智谱AI集成](ZHIPU_AI_INTEGRATION.md)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

## 👤 作者

- **quitedob** - [GitHub](https://github.com/quitedob)
- 📧 Email: dobqop999@gmail.com

---

⭐ 如果这个项目对你有帮助，请给一个 Star！
