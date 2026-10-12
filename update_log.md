# 更新日志

## UL-20261013-001 Backend 基础能力实现（2026-10-13 01:43:50 CST）

- 修改代码：补充 `backend/config/config.yaml` 运行配置和占位密钥；接入 GoFrame MySQL/Redis 驱动、统一 HTTP 响应、JWT 签发与认证、JSON/urlencoded/multipart 表单 SM2 解密、敏感字段 AES-GCM 加解密、逐消息 JWT 与 SM2 加密的 WebSocket、进程内有界队列、MySQL 搜索适配、Redis 缓存封装及本地目录存储；新增健康检查和联调 token 接口；更新 Kubernetes develop 配置挂载。
- 修改文档：完善根目录 `readme.md` 的 Backend 配置、启动方式、接口、加密协议和替代组件说明。
- 可能影响：HTTP POST/PUT/PATCH 除 token 联调入口外均要求 `data` 字段携带 SM2 密文；受保护路由要求有效 JWT；WebSocket 握手和每条消息都校验 JWT，消息 `data` 采用 SM2 密文；业务数据库写入敏感字段时需显式使用 AES helper；MySQL 搜索接口会对指定表列执行 `LIKE` 查询；Redis 缓存调用依赖可用 Redis；本地存储写入受目录配置与 feature flag 控制；Kubernetes 部署需替换 ConfigMap 中的数据库连接和密钥占位值。进程内队列不保证持久化，服务重启会丢失尚未处理任务。
- 维护注意：`security.jwt.allowDemoIssue` 默认关闭；演示 token 接口没有账号认证，只能在本地联调时临时启用。SM2/AES/JWT 占位密钥不可用于生产。搜索的 `redis` driver 当前预留给业务索引适配，内置实现为 MySQL。

## UL-20261013-002 完善 backend 运行配置（2026-10-13 01:47:37 CST）

- 修改代码：补全 `backend/config/config.yaml` 的 HTTP 超时、请求头上限、日志、请求体与 WebSocket 消息大小、MySQL 连接池与查询超时、Redis 连接池、JWT、SM2、AES 及替代组件说明；请求体和 WebSocket 限制现在从配置读取；同步 Kubernetes develop ConfigMap 中的配置，并使用集群内 MySQL/Redis 服务地址。
- 可能影响：HTTP 长连接维持不受读写超时影响；请求体和 WebSocket 单条消息大小由配置限制；数据库/Redis 最大连接数和超时时间影响连接池行为及查询等待；Kubernetes 部署会按 ConfigMap 使用 `mysql:3306`、`redis:6379` 和 `/app/storage`。替换密钥或 DSN 时需同步本地与部署配置。

## UL-20261013-003 新增管理员初始化命令（2026-10-13 01:51:44 CST）

- 修改代码：新增 `backend/internal/service/user/user.go` 的 `users` 表创建与管理员插入逻辑、`backend/internal/cmd/create_admin.go` 的 `create-admin` GoFrame 子命令、`backend/manifest/sql/001_create_users.sql` 建表文件，并在 `backend/go.mod` 增加终端密码读取和 bcrypt 依赖。命令从终端隐藏读取并二次确认密码，bcrypt 哈希后存储，重复用户名由唯一索引拒绝。
- 修改文档：在根目录 `readme.md` 增加管理员创建用法、字段说明和数据库权限要求。
- 可能影响：运行 `go run . create-admin --username <name>` 会连接配置中的 MySQL，自动创建 `users` 表并插入 `role=admin`、`status=1` 的用户；数据库账号需有建表与插入权限。命令仅能在交互式终端使用，密码长度限制为 12 至 72 字节。

## UL-20261013-004 新增根目录 Git 忽略规则（2026-10-13 02:28:43 CST）

- 修改代码：新增根目录 `.gitignore`，覆盖本机环境变量、日志、Node/前端构建产物、Go 构建输出、backend 本地存储及根目录 IDE 文件；保留 `.env.example` 类示例文件。
- 可能影响：Git 将不再显示匹配的本地文件和构建产物为未跟踪文件；各子目录原有 `.gitignore` 继续管理其专属规则，`backend/config/config.yaml` 仍由 backend 子规则忽略。
