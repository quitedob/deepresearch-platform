# 前后端审查问题修复

## 请求与范围

用户在前后端审查后要求“开始修复”。本次处理 `frontend-backend-code-review.md` 中已确认的 8 项问题，并修复验证这些路径时发现的 SSE 帧解析和过期配额并发覆盖问题。

使用 codereviewer、frontenddebugger、systematic-debugging、test-driven-development 和 verification-before-completion；浏览器验证使用 Chrome DevTools MCP。本次无需在线研究。

开始时工作区已有大量未提交改动，索引为空。先保存工作区基线，再按修复增量构造暂存内容。纳入与本次修复直接相关的既有 SSE 适配、配额行锁和少量编译修复；其余既有改动及未跟踪文件继续留在工作区，没有整体暂存。

## 修复内容

1. **前端启动与构建**：从可读历史版本恢复 store 中损坏的文本、字符串和被注释吞掉的逻辑，保留后续日志、会话置顶和批量删除功能；修正 LoadingSpinner 语法、重复声明、缺失导入及阻断 lint 的问题。
2. **题目会话归属**：在模型调用、搜索及持久化之前验证身份和目标会话所有者。未登录返回 401，非所有者返回 403，会话不存在返回 404，存储错误单独处理。
3. **研究请求契约**：三个前端入口停止发送后端不支持的逐请求模型/工具配置，界面明确说明研究使用服务器配置；共享研究 API 保留幂等键生成。
4. **过期会员扣额**：在配额行锁事务中先处理会员到期，再选择和扣减免费配额；到期时刻也视为过期。会员信息读取降级时重新加锁读取，避免用旧数据覆盖刚扣减的配额；相关配额读改写路径使用有效的 GORM 行锁。
5. **CSP 下代码复制**：移除代码块内联 onclick，通过 Vue 事件委托处理复制和引用点击；复制失败回退清理临时文本框。保留既有 CSP。
6. **SSE 断线恢复**：订阅后读取持久化终态，重放报告、证据和失败/取消状态；轮询终态覆盖其他进程完成任务的情况。每个客户端独立订阅，发送与关闭共用锁，断开时清理订阅；API 克隆共享事件元数据。前端使用 Authorization 请求头连接，处理意外 EOF 和重连，修正 Gin `data:` 无空格格式的解析。
7. **禁用账户登录状态**：三个 HTTP 客户端和 SSE 共用鉴权失败处理；401 或明确的账户禁用 403 清除两种存储中的 access token、refresh token 和用户信息，跳转登录页。普通权限/配额 403 保留登录状态。
8. **内部 Go 编译**：遗留 llm 类型复用已有 provider 类型；移除无人引用且没有实现的三个失效 service 接口包；修复 tools 属性类型声明。不新增 AI 框架或模型适配。

新增题目会话权限、过期配额及并发读取、SSE 终态重放/多客户端/取消/并发关闭、前端请求与鉴权契约测试。修正原有任务租约测试使用固定历史时钟导致新建任务尚不可领取的问题，使用当前时间并对齐 PostgreSQL 微秒精度。

## 验证结果

| 检查 | 结果 |
| --- | --- |
| 根模块 `go test -count=1 ./src/cmd/server` | PASS；server 包无测试，验证入口及依赖编译 |
| 内部模块 `go test -count=1 -p 1 ./...` | PASS；工作区与独立导出的暂存版本均通过 |
| PostgreSQL 集成测试 | PASS；显式配置专用本地 `audit_fixes_test` 数据库，覆盖会话归属、会员到期、并发扣额与任务持久化；未访问业务数据库 |
| 前端 `npm test` | PASS，8 项测试 |
| 前端 `npm run lint:check` | PASS |
| 前端 `npm run build` | PASS；工作区与独立暂存版本均验证 |
| Chrome DevTools 研究流程 | PASS；真实研究提交 API 返回 201，首次 SSE 主动断开后重连，页面显示已保存报告和研究证据 |
| Chrome DevTools 复制与账户禁用 | PASS；CSP 下点击/键盘触发代码复制，复制参数正确且无内联事件；profile 403 后清除六项存储值并返回登录页 |
| `git diff --cached --check` | PASS |

浏览器回归采用独立上下文和本地验证服务：身份、研究生成结果及无关列表接口为测试夹具，研究提交 HTTP handler、DAO、PostgreSQL 持久化和 SSE 服务使用实际代码。Clipboard API 的写入目标被 spy 替换，验证复制文本与调用，不修改系统剪贴板。检查了可访问性快照、控制台、网络、键盘复制和窄窗口提示可见性；浏览器最小窗口宽度实际为 501px，没有宣称完成 390px 设备验收。

暂存版本独立导出到 `build/temp/audit-fixes-staged-check/` 后执行检查，以避免依赖其他未提交文件才能通过。首次独立检查暴露的 HealthCheck 导入、SearchInterface 未使用导入、PaperView 重复函数和 tools 声明问题已按最小范围纳入修复，并重新验证。

## 证据与限制

- `build/temp/audit-fix-baseline/manifest.json`：原始工作区范围和基线备份索引。
- `build/temp/audit-fixes-browser-evidence.json`：浏览器快照、研究提交/重连网络及复制和账户禁用验证结果。
- `build/temp/audit-fixes-staged-go-final.log`、`build/temp/audit-fixes-workspace-go-final.log`：最终两套内部模块测试输出。
- `build/temp/audit-fixes-final-index-check.json`：暂存源码与已验证导出的内容一致性及文件行数检查。
- DevTools 直接写证据文件被其工作区路径限制拒绝，改为保存工具返回的验证记录；没有替换要求的浏览器工具。
- 未调用真实外部 LLM，也未执行生产部署；本次浏览器结果不能代替真实模型服务的端到端验收。
- Go race 检查未运行成功：环境为 `CGO_ENABLED=0`，没有可用的 gcc/clang。普通并发回归测试通过。
- 既有 Vite 分块/混合动态导入警告以及 HTML meta CSP 的 frame-ancestors 无效警告仍存在。
- 原审查中更大范围的插件初始化位置、业务分层等低优先级架构建议未在本次缺陷修复中重构。
- 本次启动的验证服务、浏览器标签和隔离 PostgreSQL 在收尾时关闭；临时证据保留于已忽略的 `build/temp/`，不纳入提交。
