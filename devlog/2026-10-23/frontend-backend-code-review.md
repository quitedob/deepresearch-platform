# 前后端代码审查

## 请求与范围

- 请求：启用 `codereviewer`，扫描前端与后端；将 `@fronend` / `@backend` 分别解释为 `vue/` / `src/`。
- 审查基线：`136db96`，分支 `feat/durable-research-submission`；审查对象为当前工作区，包含已有 66 个修改文件及 9 个未跟踪的后端源文件/测试文件。`.codex/` 不在应用审查范围内。
- 使用 `codereviewer`、`frontenddebugger` 与 Chrome DevTools MCP。未修改应用源码或测试；依据仓库 AGENTS.md 的明确要求新增本开发日志。
- 完成全量文件结构盘点，并重点追踪启动、鉴权、题目会话归属、研究提交/配额/工作器/SSE、论文持久化与前端 API 消费路径。属于风险导向扫描，未逐行证明所有模块正确。
- 结论：审查完成，存在实质风险；不代表应用通过验收。以下 8 项均有运行结果或完整静态调用链支持；相对基线，1 项由现有未提交改动引入，7 项已经存在。

## 已确认问题

### 1. HIGH / PRE_EXISTING：前端入口依赖存在语法错误

`vue/src/store/index.js:30` 的字符串缺少结束引号；同文件第 79、865 行也有损坏文本。`App.vue`、`Home.vue` 引用该 store，Vite 生产构建在第 30 行报 `Unterminated string constant`，开发页面请求该模块返回 HTTP 500 并显示 Vite 错误覆盖层，正常页面无法启动。

建议逐处修复损坏的字符串与编码，再运行 lint、构建及页面验证。对照 `git show HEAD:vue/src/store/index.js` 确认该问题已在基线存在。

### 2. HIGH / PRE_EXISTING：生成题目可写入其他用户的会话

`src/internal/api/v1/ai_question.go:278` 开始直接使用请求提供的 `session_id` 创建消息与题目，并在第 322 行读取、更新该会话的计数。该函数没有验证 `session.UserID == userID`；DAO 的写入也没有所有者条件。路由上的 `middleware.Auth()` 只验证调用者身份，无法替代资源归属校验。

前提是已认证调用者知道另一个用户的有效会话 ID，且模型成功返回题目；随后可污染该会话。与同文件 `GetSession` 第 650 行和 `SaveQuestionsToSession` 第 845 行的明确所有权检查不一致。建议在模型调用及任何写入前验证归属，并将相关写入放入一致的事务。此结论来自静态调用链，未访问真实用户数据或调用收费模型。

### 3. HIGH / PRE_EXISTING：主界面的研究请求与后端契约冲突

`vue/src/views/Home.vue:451` 的请求总是携带非空 `llm_config.provider/model`，但 `src/internal/api/v1/research.go:132` 明确对这些字段返回 HTTP 400、`UNSUPPORTED_RESEARCH_CONFIGURATION`，因此该研究入口无法提交成功。前端 `stores/research.js:114` 也构造了同类配置。

建议统一当前支持的请求契约：前端停止发送不支持的模型/工具选项，或完整实现后端的逐请求配置。现有幂等键生成不会消除该冲突。前后端冲突代码均已存在于基线；受前端启动故障影响，未宣称完成真实研究流程的浏览器端到端验证。

### 4. HIGH / PRE_EXISTING：研究扣额没有验证高级会员到期时间

`src/internal/repository/dao/research_submission_dao.go:288` 仅按 `MembershipType` 选择高级配额并按周期重置，没有检查 `ExpiresAt`。正式 `/research/start` 路径直接经过提交 DAO 调用此函数，不经过会处理会员过期的 `MembershipDAO.GetOrCreateMembership`。

因此，已到期但数据库仍标为 `premium` 的活跃账户，在未先触发会员查询降级时，仍可通过直接提交研究继续使用高级配额；周期重置也会继续提供高级额度。建议在同一行锁事务中处理会员有效期，再选择并扣减配额。现有定时重置代码也没有会员过期降级逻辑。数据库集成复现未执行。

### 5. MEDIUM / INTRODUCED：新增 CSP 阻止代码块复制按钮

现有工作区在 `vue/index.html:12` 新增 `script-src 'self'`，而 `vue/src/components/MessageItem.vue:457` 和第 460 行仍生成带内联 `onclick` 的复制按钮。

Chrome DevTools 中隔离挂载真实 `MessageItem.vue`，传入本地代码块样例后点击代码块的“复制”，浏览器明确报内联事件处理器违反 CSP，动作被阻止。建议保留 CSP，改用 Vue 事件或受控的事件委托处理复制；不要通过放开所有内联脚本来绕过。该验证是组件级验证，不是正常应用页面流程。

### 6. MEDIUM / PRE_EXISTING：研究 SSE 重连不补发终态

`src/internal/service/research_service.go:397` 的 `StreamProgress` 仅确认会话存在，随后总是调用 `EventStream.CreateStream` 创建空通道。任务完成后完成事件已发送且流被关闭；这时重连会得到新空通道，没有从数据库恢复完成状态与结果。

`vue/src/views/Home.vue:603` 有实际重连分支。如果任务恰好在断线期间完成，重连后只会收到连接/心跳消息，直到超时仍不能显示最终报告。建议订阅时读取持久化终态并发送结果，或在前端重连时查询状态/导出结果，并处理订阅与完成之间的竞态。该判断为静态路径验证。

### 7. MEDIUM / PRE_EXISTING：主 API 客户端未清理禁用账户的登录状态

后端 `src/internal/middleware/auth.go:109` 返回 HTTP 403、`ERR_ACCOUNT_INACTIVE`；主客户端 `vue/src/api/index.js:73` 只在 401 时清理凭据。`vue/src/utils/apiClient.js` 的现有改动修复了另一个客户端，但 `api/user.js` 和聊天/研究 API 仍使用 `api/index.js`。

在浏览器中调用真实主客户端响应拦截器，使用不发网络请求的 Axios adapter 模拟上述 403，确认 access token 与 refresh token 均保留。路由守卫第 138 行又会将携带 token 的登录页访问重定向到 `/home`。建议在实际使用的客户端处理该业务错误，并统一公共鉴权处理。模拟数据已恢复，未使用真实凭据。

### 8. MEDIUM / PRE_EXISTING：内部 Go 模块无法通过全量编译检查

`go test -count=1 ./...` 在 `src/internal/` 失败：

- `pkg/llm/factory.go:35` 等 provider 的 `Generate` 使用 `provider.Message`，不满足需要 `llm.Message` 的接口；`pkg/llm/types.go:5` 还有未使用 import。
- `service/chat/interface.go:16` 等位置引用不存在的请求/响应类型。
- `service/llm/interface.go`、`service/research/interface.go` 同样引用不存在的 DTO。

相关文件与 HEAD 内容一致。这些遗留包目前不阻止正式 server 入口编译，但阻断内部模块的完整测试门禁。建议按照 Eino 唯一框架要求收敛遗留抽象，或先修正仍需保留的接口契约；不要仅凭根模块检查通过认定两个模块均通过。

## 结构与复用检查

- 按 LF/CRLF 物理行计数，盘点 255 个第一方 `.go`、`.js`、`.vue`、`.css` 文件，共 77,015 行；其中后端 164 个、前端 91 个，包含 17 个 Go 测试文件。排除依赖、构建输出、锁文件、图片和数据文件。
- 没有文件达到 2,000 行。最大文件为 `vue/src/components/ContentModeration.vue`，1,971 行；因此没有触发强制逐文件大文件拆分分析。
- 前端为 Vue 3/Vite/Pinia/Router，常规链路为视图/组件到 API 适配层。顶层 components 有 33 个文件，views 有 15 个文件；单凭数量不构成缺陷。
- LOW：`api/index.js`、`utils/apiClient.js`、`services/api.js` 并存独立 HTTP 客户端，已产生第 7 项的鉴权修复遗漏。建议先抽取共享 token/error 处理，再保持各调用方的响应解包契约逐步迁移；验证 401、账户禁用 403、普通权限 403、SSE 断开，避免把所有 403 都当作登出。
- 后端正式入口为 `cmd/server/main.go -> api/router.go -> api/v1`；同时保留 `handler -> service -> repository` 路径及不能编译的遗留 service 接口。正式聊天/题目 API 仍直接编排模型、配额及 DAO，重复行为分散在服务层和 API 层。
- LOW：`cmd/server/llm_init.go`、`zhipu_client.go` 及 main 内的 provider 注册与仓库要求的 `pkg/eino` 插件边界不符。建议先将 provider 构造/鉴权适配与注册收敛到 `pkg/eino/model` 和导出的初始化入口，再将聊天业务编排收敛到共享服务；main 保留配置和依赖装配，API 保留传输层校验。应保留流取消、配额、归属及错误映射语义。此次仅分析，没有执行重构。
- 排除的候选：Markdown HTML 渲染存在危险入口，但当前 CSP 会阻断所考虑的内联脚本载荷，未报告已复现 XSS；`buildToolArgs` 的 zread 分支缺少 query，但当前策略映射不选择该工具，未把不可达分支报告为现行流程故障。

## 验证结果与限制

| 检查 | 结果 |
| --- | --- |
| 根模块 `go test -count=1 ./src/cmd/server` | PASS：入口及依赖编译通过，该包没有测试 |
| 内部模块 `go test -count=1 ./...` | FAIL：上述 4 个遗留包编译失败；其余已执行的测试包通过 |
| 数据库集成测试 | NOT_RUN：本轮命令将 `TEST_DATABASE_URL` 置空，仅对该进程有效，集成测试按现有守卫跳过；未配置隔离数据库 |
| `npm run lint:check` | FAIL：27 errors，含 store 字符串及 `LoadingSpinner.vue:140` 语法错误和其他静态错误 |
| Vite 生产构建（`build({build:{write:false}})`） | FAIL：store 第 30 行语法错误；使用现有配置，在内存构建，不写 dist |
| Chrome DevTools 页面检查 | 已收集可访问性快照、控制台与网络；store 请求 500，页面显示 Vite 解析错误 |
| Chrome DevTools 隔离组件检查 | 已复现代码块复制被 CSP 阻止 |
| Chrome DevTools API 拦截器检查 | 已复现模拟账户禁用 403 后 token 保留；adapter 未向后端发送请求 |
| `gofmt -l` | 96/164 个 Go 文件需要格式化；仅检查，没有改写 |
| 当前工作区 `git diff --check` | PASS |

浏览器环境为 Windows / Chrome 155，本地 Vite 服务地址为 `http://127.0.0.1:3000/`，使用独立浏览器上下文。页面因语法错误不能进入正常业务流程，因此登录、真实研究/论文、管理员操作、响应式与完整键盘流程没有端到端验收。未启动会自动迁移数据库的后端，也未执行真实 LLM 调用或外部在线研究。

验证证据保留在本次会话的工具输出中，没有另外生成截图、快照文件或临时报告。隔离组件已卸载、模拟存储已恢复；审查启动的浏览器标签及 Vite 服务在收尾时关闭。提交范围仅为本日志，既有源码与测试改动继续保留在工作区。
