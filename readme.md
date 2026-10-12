这是一个全栈项目模板，当前包含 GoFrame backend 脚手架。前端 app 可按项目需要接入。

## Backend

### 技术与组件

- GoFrame v2 HTTP 服务，默认监听 `:8000`。
- MySQL 与 Redis 通过 GoFrame 驱动注册，连接参数统一写在 `backend/config/config.yaml`。
- RabbitMQ 功能由进程内有界队列替代，不依赖外部 RabbitMQ。队列满时明确返回错误；进程重启后未处理任务不会保留。
- Elasticsearch 功能由 MySQL 参数化 `LIKE` 查询适配器替代；Redis 可用于缓存或由业务维护独立搜索索引。
- COS 功能由配置目录下的本地文件存储替代，存储 key 会拒绝目录穿越路径。
- `features.rabbitmq.enabled`、`features.search.enabled`、`features.cos.enabled` 控制对应能力是否启动/可用。

### 配置与启动

复制或修改 `backend/config/config.yaml`，填写 MySQL、Redis、JWT、SM2 与 AES 配置。密钥目前均为占位内容，不可用于生产。JWT secret 应替换为随机 32 字节以上密钥；AES key 应设置为 16、24 或 32 字节；SM2 使用匹配的 PEM 公私钥。

```sh
cd backend
go run .
```

部署到 Kubernetes 时，develop overlay 将 ConfigMap 挂载为 `/app/config/config.yaml`。配置中的示例数据库口令和密钥需在部署前替换。

### 用户表与管理员初始化

执行以下命令创建管理员。命令会先确保 MySQL 中的 `users` 表存在，再提示输入并确认密码；密码不会回显或作为命令行参数传递。

```sh
cd backend
go run . create-admin --username admin
```

用户名需为 3 到 64 位字母、数字、点、下划线或连字符；密码需为 12 到 72 字节，并使用 bcrypt 哈希后保存。若用户名已存在，命令会失败且不会覆盖账号。数据库账号需要拥有 `CREATE TABLE` 和 `INSERT` 权限。建表 SQL 见 `backend/manifest/sql/001_create_users.sql`。

用户表字段包括自增 `id`、唯一 `username`、`password_hash`、`role`、`status`、`created_at` 和 `updated_at`；管理员的 `role` 为 `admin`，初始状态为启用。

### HTTP 接口与响应

- `GET /health`：健康状态。
- `GET /hello`：示例接口。
- `POST /api/v1/auth/token`：本地联调 token 示例，默认由 `security.jwt.allowDemoIssue: false` 禁用。开启后可提交 `{"subject":"user-id"}`；正式项目必须替换为实际账号认证逻辑。
- `GET /ws?token=<JWT>`：建立 WebSocket 连接；也可在握手时使用 `Authorization: Bearer <JWT>`。
- 其他 HTTP 路由默认要求 `Authorization: Bearer <JWT>`。

普通 HTTP JSON 响应统一为 `{ "code": 0, "message": "OK", "data": ... }`，错误时 `code` 为非零值。POST/PUT/PATCH 的 JSON 请求按 `{ "data": "<Base64 SM2 ciphertext>" }` 发送；urlencoded 与 multipart 表单也必须提供加密的 `data` 字段。服务端解密后将明文 JSON 放回 `data`，业务处理器只读取解密后的 `data`。token 示例接口是建立登录态的明文例外。请求体上限为 8 MiB。

AES 敏感字段通过 `backend/internal/service/crypto` 的 `EncryptSensitive` / `DecryptSensitive` 显式加解密，格式为 `aes:v1:<Base64(nonce+ciphertext)>`。业务写数据库前必须主动调用加密函数，读取时调用解密函数；模板不猜测哪些列属于敏感数据。

### WebSocket 消息协议

每条客户端消息都必须包含当前 JWT，`data` 只放 SM2 加密后的 Base64 内容：

```json
{"token":"<JWT>","event":"ping","data":"<Base64 SM2 ciphertext>"}
```

解密后的 `data` 必须是 JSON。服务端成功响应沿用 `{code,message,data}`，其中响应 `data` 是用配置中的 SM2 公钥加密后的 Base64 字符串；连接握手和每条消息都会校验 JWT，消息 token 的 subject 必须与握手用户一致。
