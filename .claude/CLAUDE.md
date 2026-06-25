### claude.md : FrostAgent Memory MCP (Identity & Admin UI 增强版)

**项目上下文 (Project Context)**
当前开发目标是为 FrostAgent（一个基于 Golang 的角色扮演 AI 编排服务 ）开发一套增强版 Memory MCP。核心诉求包括：基于 QQ 号的跨群组实体识别、多维度的用户惯用名学习、支持框架下发的持久化记忆工具，以及一个用于人工干预的 Web Admin UI。

**核心架构约束 (Architecture Rules)**

* **双轨协议**: 进程需同时维护两套接口，即面向大模型的 MCP JSON-RPC 接口（用于记忆读写），以及面向管理员的 HTTP RESTful API（用于前端页面 CRUD）。
* **数据隔离**: 持久层 SQLite 必须以 QQ 号（即 UserID）作为绝对主键，群号（GroupID）作为联合维度。

#### Phase 1: 实体识别与数据模型搭建 (Data Modeling)

* 初始化 SQLite 数据库，设计 `Users` 核心表，以 QQ 号作为全局唯一键。
* 建立 `UserAliases` 关联表，字段包含 QQ_ID, Group_ID, Called_Name，用于存储用户在不同群聊中的惯用名或群名片。
* 建立 `LongTermFacts` 关联表，存储由 AI 提取的用户特征（如喜好、人际关系），支持按分类（Category）打标签。

#### Phase 2: MCP 工具扩展与 Harness 注入设计 (Agent Capabilities)

* 开发 `query_user_profile` 工具：输入 QQ 号与当前群号，返回该用户的全局特征以及在该群的特定称呼。
* 开发 `learn_user_alias` 工具：当 AI 在对话中捕捉到“别人怎么称呼该用户”或“用户要求更改称呼”时，调用此工具更新 `UserAliases` 表。
* 开发 `extract_and_store_fact` 工具：用于提取结构化的长期事实。
* **Prompt 注入规范**: 提供一份针对 FrostAgent/AstrBot 主控节点的 System Prompt 模板，明确指示大模型：“当对话中出现新人物或称呼变化时，必须强制调用对应 MCP 工具进行记忆沉淀”。

#### Phase 3: 管理员后台看板开发 (Admin Web UI)

* 在 Golang 服务中引入轻量级 Web 框架（如 Gin 或 Fiber），监听独立端口（如 `:8080`）。
* 开发一套基础的 RESTful API：`/api/users`、`/api/aliases`、`/api/facts`，支持完整的增删改查。
* 利用 Golang 的 `embed` 特性，内嵌一个单文件的前端页面（原生 HTML + TailwindCSS + Vue/Alpine.js），实现开箱即用的可视化记忆分类展示板，无需额外部署前端项目。

#### Phase 4: 验收标准 (Definition of Done)

| 验收维度 | 具体完成标准 |
| --- | --- |
| **实体跨群识别** | AI 在群 A 记录了用户的外号，用户在群 B 交互时，系统能准确识别其身份，且不会混淆群 A 和群 B 的特定称呼。 |
| **指令遵循与学习** | 在对话中发送“以后在这个群请叫我舰长”，AI 能够准确调用工具写入 SQLite，且后续回复立刻生效。 |
| **看板可用性** | 浏览器访问本地管理端口，页面能以卡片或列表形式清晰展示各 QQ 号对应的身份数据，且通过页面修改数据后，大模型的下一次回复能体现出修改。 |
| **架构稳定性** | MCP 接口高频被大模型调用的同时，管理员在 Web 页面进行数据删除操作，服务不发生锁死或竞态崩溃（Panic）。 |