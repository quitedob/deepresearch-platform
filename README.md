# Go Deep Research

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Vue-3.x-4FC08D?style=flat&logo=vue.js" alt="Vue Version">
  <img src="https://img.shields.io/badge/CloudWeGo-Eino-orange" alt="Eino Framework">
  <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg" alt="PRs Welcome">
</p>

基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) 框架构建的新一代 AI 研究平台，集成多智能体协作、自主研究工作流、智能题目生成和完整的会员管理体系。

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

### 📄 AI论文生成系统
- **多模板支持**: APA、MLA、Chicago、IEEE等学术格式
- **章节生成**: 摘要、引言、文献综述、方法论、结果、讨论、结论
- **智能引用**: 自动生成符合规范的引用格式
- **字数统计**: 实时统计各章节字数
- **章节重生成**: 支持单独重新生成某个章节
- **导出功能**: 支持Markdown和PDF格式导出

### 📝 AI题目生成系统
- **支持题型**: 单选题、多选题、判断题、简答题
- **智能生成**: 自动生成题目ID、分值、解析
- **难度分级**: 简单/中等/困难
- **知识点标注**: 自动标签和分类
- **会话管理**: 独立的出题会话系统
- **防重复**: 智能避免重复题目

### 👥 会员与配额系统
| 用户类型 | 聊天配额 | 研究配额 | 重置周期 |
|----------|----------|----------|----------|
| 普通用户 | 10次/天 | 1次/天 | 每天 |
| 高级会员 | 50次 | 10次 | 5小时 |

**激活码系统**:
- 批量生成和管理激活码
- 使用记录追踪
- 自定义配额配置

### 🔔 通知系统
- 配额不足提醒
- 系统更新通知
- 激活成功通知
- 研究完成通知
- 实时未读统计

### 🔌 MCP工具集成
- **WebSearchTool** - 智谱AI网络搜索
- **WebSearchPrime** - 增强网络搜索
- **ArxivTool** - 学术论文搜索
- **WikipediaTool** - 维基百科查询
- **ZReadTool** - GitHub代码阅读
- **WebReaderTool** - 网页内容提取
- **ReliabilityTool** - 可靠性评估

### 🛠️ 管理后台
- **用户管理**: 状态管理、会员设置、配额分配
- **模型配置**: 动态配置模型参数
- **激活码管理**: 生成、查询、批量操作
- **通知管理**: 系统通知发布
- **聊天记录**: 查询和导出用户对话
- **统计面板**: 实时数据和可视化

## 🏗️ 技术架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Vue 3 Frontend                               │
│         (Vite + Pinia + Vue Router + Element Plus)                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │ ChatContainer│  │ResearchSpace │  │  AISpace     │             │
│  │ PaperGen     │  │MembershipMgr │  │  Notification │             │
│  │ AdminPanel   │  │ModelConfig   │  │  Monitoring  │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
└─────────────────────────────────────────────────────────────────────┘
                              ↕ HTTP/SSE/WebSocket
┌─────────────────────────────────────────────────────────────────────┐
│                      Go Backend (Gin)                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │ Auth API     │  │ Chat API     │  │ Research API │             │
│  │ Membership   │  │ AIQuestion   │  │ Paper API    │             │
│  │ Notification │  │ MCP API      │  │ LLM API      │             │
│  │ Admin API    │  │ MCP API      │  │ LLM API      │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
└─────────────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────────────┐
│                    Eino Component Layer                             │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              Multi-Agent Orchestration                       │   │
│  │  Planner → Research → Evaluator → Critic → Report           │   │
│  └─────────────────────────────────────────────────────────────┘   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │ LLMScheduler │  │ ChatModel    │  │ Tool Registry│             │
│  │ StreamManager│  │ ConfigMgr    │  │ QuotaManager │             │
│  │ PaperGen     │  │ CitationMgr  │  │ TemplateMgr  │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
└─────────────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────────────┐
│                   Infrastructure Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │ PostgreSQL   │  │ Redis        │  │ LLM APIs     │             │
│  │ (Users/Chat  │  │ (Cache/      │  │ DeepSeek     │             │
│  │  Research/   │  │  Session)    │  │ Zhipu/GLM    │             │
│  │  Papers/...) │  │              │  │ OpenAI/...)  │             │
│  │  Quota/...)  │  │              │  │ OpenAI/...)  │             │
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
cp .env .env
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
deepresearch-platform/
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

## 📡 API 端点

### 🔐 认证 (`/api/v1/auth`)
| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/register` | 用户注册 |
| POST | `/login` | 用户登录 |
| POST | `/refresh` | 刷新Token |
| POST | `/logout` | 用户登出 |

### 👤 用户管理 (`/api/v1/user`)
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/profile` | 获取用户信息 |
| PUT | `/profile` | 更新用户信息 |
| GET | `/preferences` | 获取用户偏好 |
| PUT | `/preferences` | 更新用户偏好 |
| POST | `/change-password` | 修改密码 |
| GET | `/stats` | 获取用户统计 |
| DELETE | `/delete-account` | 删除账户 |
| GET | `/memory-settings` | 获取记忆设置 |
| PUT | `/memory-settings` | 更新记忆设置 |

### 💬 聊天功能 (`/api/v1/chat`)
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/models` | 获取可用模型列表 |
| POST | `/sessions` | 创建聊天会话 |
| GET | `/sessions` | 获取会话列表 |
| GET | `/sessions/:id` | 获取会话详情 |
| PUT | `/sessions/:id` | 更新会话 |
| DELETE | `/sessions/:id` | 删除会话 |
| GET | `/sessions/:id/messages` | 获取消息列表 |
| DELETE | `/sessions/:id/messages` | 清空消息 |
| POST | `/chat` | 发送消息（非流式） |
| POST | `/chat/stream` | 流式聊天 |
| POST | `/chat/web-search` | 联网搜索聊天 |
| GET | `/sessions/:id/context-status` | 获取上下文状态 |
| POST | `/sessions/:id/summarize-and-new` | 总结并新建会话 |

### 🔬 深度研究 (`/api/v1/research`)
| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/start` | 启动研究 |
| GET | `/stream/:session_id` | 流式获取研究进度（SSE） |
| GET | `/status/:session_id` | 获取研究状态 |
| GET | `/sessions` | 获取研究会话列表 |
| GET | `/export/:session_id` | 导出研究结果 |
| GET | `/search` | 搜索研究记录 |
| GET | `/statistics` | 获取研究统计 |

### 📄 论文生成 (`/api/v1/paper`)
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/templates` | 获取论文模板列表 |
| GET | `/citation-styles` | 获取引用样式列表 |
| POST | `/start` | 开始生成论文 |
| GET | `/status/:id` | 获取论文生成状态 |
| GET | `/result/:id` | 获取论文内容 |
| GET | `/export/:id` | 导出论文 |
| GET | `/list` | 获取论文列表 |
| DELETE | `/:id` | 删除论文 |
| POST | `/regenerate` | 重新生成章节 |
| GET | `/stream/:id` | 流式获取生成进度 |

### 📝 AI题目生成 (`/api/v1/ai`)
| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/generate-questions` | 生成题目 |
| POST | `/question-sessions` | 创建出题会话 |
| GET | `/question-sessions` | 获取出题会话列表 |
| GET | `/question-sessions/:id` | 获取出题会话详情 |
| PUT | `/question-sessions/:id` | 更新出题会话标题 |
| DELETE | `/question-sessions/:id` | 删除出题会话 |
| POST | `/question-sessions/:id/questions` | 保存题目到会话 |
| GET | `/question-config` | 获取出题配置 |

### 👥 会员系统 (`/api/v1/membership`)
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/` | 获取会员信息 |
| GET | `/quota` | 获取配额信息 |
| POST | `/activate` | 激活码激活 |
| GET | `/check-chat-quota` | 检查聊天配额 |
| GET | `/check-research-quota` | 检查研究配额 |

### 🔔 通知系统 (`/api/v1/notifications`)
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/` | 获取用户通知 |
| GET | `/unread-count` | 获取未读数量 |
| POST | `/:id/read` | 标记为已读 |
| POST | `/read-all` | 全部标记已读 |

### 🔌 MCP工具 (`/api/v1/mcp`)
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/tools` | 获取可用工具列表 |
| GET | `/tools/:tool_name` | 获取工具详情 |
| POST | `/tools/call` | 调用工具 |

### 🤖 LLM管理 (`/api/v1/llm`)
| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/providers` | 获取LLM提供商列表 |
| GET | `/models` | 获取所有可用模型 |
| GET | `/metrics` | 获取LLM指标 |
| POST | `/test` | 测试LLM提供商 |

### 🛠️ 管理员功能 (`/api/v1/admin`)
| 分类 | 方法 | 端点 | 描述 |
|------|------|------|------|
| 统计 | GET | `/stats` | 获取管理员统计 |
| 用户 | GET | `/users` | 列出用户 |
| | PUT | `/users/:id/status` | 更新用户状态 |
| | PUT | `/users/:id/membership` | 更新用户会员 |
| | PUT | `/users/:id/quota` | 设置用户配额 |
| | POST | `/users/:id/reset-quota` | 重置用户配额 |
| | PUT | `/users/batch-status` | 批量更新状态 |
| 聊天记录 | GET | `/users/:id/chat-history` | 获取聊天历史 |
| | GET | `/users/:id/chat-history/export` | 导出聊天历史 |
| 激活码 | GET | `/activation-codes` | 列出激活码 |
| | POST | `/activation-codes` | 创建激活码 |
| | GET | `/activation-codes/:id` | 获取激活码详情 |
| | PUT | `/activation-codes/:id` | 更新激活码 |
| | DELETE | `/activation-codes/:id` | 删除激活码 |
| 通知 | GET | `/notifications` | 列出通知 |
| | POST | `/notifications` | 创建通知 |
| | DELETE | `/notifications/:id` | 删除通知 |
| 模型配置 | GET | `/providers` | 获取提供商配置 |
| | PUT | `/providers` | 更新提供商配置 |
| | GET | `/models` | 获取模型配置 |
| | PUT | `/models` | 更新模型配置 |
| | PUT | `/models/batch` | 批量更新模型 |
| | POST | `/models/test` | 测试模型 |
| | GET | `/models/registered` | 获取已注册模型 |
| | POST | `/models/sync` | 同步模型到数据库 |
| 配额配置 | GET | `/quota-configs` | 获取配额配置 |
| | PUT | `/quota-configs` | 更新配额配置 |
| | PUT | `/users/:id/custom-quota` | 设置用户自定义配额 |
| | PUT | `/users/batch-quota` | 批量设置配额 |

## 🎯 核心交互功能

| 功能 | 按钮标识 | 后端实现 | 说明 |
|------|----------|----------|------|
| **深度思考** | 🧠 Deep Think | `getDeepThinkingModel()` | 自动切换到推理模型（deepseek-reasoner/glm-4.7） |
| **联网搜索** | 🌐 Web Search | `ChatWebSearch()` + `WebSearchTool` | 调用智谱AI web_search 获取实时信息 |
| **深度研究** | 🔬 Deep Research | `ResearchAgent` 多智能体系统 | 规划→执行→评估→报告全流程 |
| **论文生成** | 📄 Generate Paper | `PaperAPI` + 模板系统 | 支持多种学术格式，智能引用生成 |

### 研究流程详解
```
┌─────────────────────────────────────────────────────────────┐
│  1. Planner Agent    - 分析问题，生成研究计划               │
│  2. Research Agent   - 使用工具收集信息                     │
│  3. Evaluator Agent  - 评估信息质量和相关性                 │
│  4. Critic Agent     - 批评分析，找出遗漏                  │
│  5. Report Agent     - 生成结构化报告                       │
└─────────────────────────────────────────────────────────────┘
```

### 论文生成流程
```
┌─────────────────────────────────────────────────────────────┐
│  1. 选择模板        - APA/MLA/Chicago/IEEE                 │
│  2. 输入主题        - 论文题目和研究方向                    │
│  3. 生成大纲        - 自动生成章节结构                      │
│  4. 逐章生成        - 使用 LLM 生成各章节内容              │
│  5. 智能引用        - 自动生成符合规范的引用               │
│  6. 导出论文        - Markdown/PDF 格式                    │
└─────────────────────────────────────────────────────────────┘
```

## 📚 详细文档

- [API 文档](src/docs/API_DOCUMENTATION.md) - 完整的API参考
- [架构设计](src/docs/ARCHITECTURE.md) - 系统架构和设计模式
- [开发指南](src/docs/DEVELOPER_SETUP.md) - 开发环境搭建
- [配置参考](src/docs/CONFIGURATION_REFERENCE.md) - 配置文件详解
- [部署指南](src/docs/DEPLOYMENT_SETUP.md) - 生产部署说明

## 🌟 项目亮点

1. **多智能体协作** - 基于 Eino 框架的 ReAct Agent 系统
2. **实时流式输出** - SSE 推送，支持长文本流式展示
3. **智能上下文管理** - 自动监控 token 使用，智能总结
4. **完整的会员体系** - 配额管理、激活码、自动重置
5. **AI 题目生成** - 支持多种题型，智能防重复
6. **AI 论文生成** - 多种学术格式，智能引用系统
7. **MCP 工具集成** - 可扩展的工具系统
8. **GLM Coding Plan 支持** - 专为编程优化的 API 端点

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
  -d '{"title":"测试","llm_provider":"zhipu","model_name":"glm-4.7"}'

# 3. 发送消息
curl -X POST http://localhost:8080/api/v1/chat/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"session_id":"<SESSION_ID>","message":"你好","stream":false}'

# 4. 启动深度研究
curl -X POST http://localhost:8080/api/v1/research/start \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"query":"Go语言的最新发展","mode":"deep"}'

# 5. 生成论文
curl -X POST http://localhost:8080/api/v1/paper/start \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"title":"人工智能的发展","template":"apa","citation_style":"apa"}'
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
        - glm-4.7
        - glm-4.5-air
```

### Cursor 集成 GLM Coding Plan

```
Provider: OpenAI Compatible
API Key: [你的 GLM API Key]
Base URL: https://api.z.ai/api/coding/paas/v4
Model: GLM-4.7 (大写)
```

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

- [CloudWeGo Eino](https://github.com/cloudwego/eino) - 优秀的 Go 语言 AI 应用框架
- [DeepSeek](https://www.deepseek.com/) - 提供强大的推理模型
- [智谱AI](https://www.bigmodel.cn/) - 提供 GLM 系列模型
- 所有贡献者和使用者

## 🔗 相关链接

- [Eino 官方文档](https://www.cloudwego.io/zh/docs/eino/)
- [Eino GitHub](https://github.com/cloudwego/eino)
- [Eino-Ext 组件库](https://github.com/cloudwego/eino-ext)
- [GLM Coding Plan](https://www.bigmodel.cn/glm-coding)

---

<div align="center">

**如果这个项目对你有帮助，请给一个 ⭐️ Star！**

Made with ❤️ by [quitedob](https://github.com/quitedob)

</div>
