## Why

api-gen 当前硬编码"主键统一由服务端生成"（design-v2 决策 #27）：实体级 `Create` 请求只携带各资源载荷、响应回带 key，且无配置项可改变该行为。

这对**自然主键 + 数据导入**场景是硬伤。以首个重度使用方 my-wechat 项目为例：全部 28 个实体的 key 都是源数据库已知的自然键（如 `MessageId {account_id, session_username, local_id}`，local_id 必须保持与原始表一致，排序/快照/撤回等关联数据都依赖它），导入脚本调用 `CreateMessage` 时服务端既无法生成 local_id，也无从得知 account_id / session_username——导入主流程走不通。

AIP-133 本身支持 user-specified IDs，api-gen 需要补齐这一能力。

## What Changes

### 1. `create` 配置扩展（向后兼容）

- `create: {}` / `create: { key: server }`：现状，服务端生成主键（默认）
- `create: { key: client }`：新增，客户端指定主键
- 非法取值在 validate 阶段 fail-fast

### 2. client 模式下的生成契约

- `Create<Entity>Request` 前置 key 字段：`{ <KeyType> key = 1; <各资源> = 2..N+1 }`（资源字段号顺延，均可选，部分创建语义不变）
- HTTP 路径携带 key 叶子段：`post: "/{prefix}/{Service}/{collection}/{key叶子段...}"`，`body: "*"`（path 注入 key、body 携带资源）
- 响应仍为 `{ <KeyType> key = 1 }`（client 模式下为回显），两种模式响应形态统一

### 3. 文档

- README.md / README_EN.md 的 `create` 配置说明（两种模式对照）
- design-v2.md 决策 #27 修订（"无 generates 配置" → `create.key` 策略位落地）

## Impact

### 受影响的代码

- `internal/yaml/parser.go`：`Entity.Create *struct{}` → `*CreateDef{ Key string }`
- `internal/yaml/validate.go`：新增 `create.key` 取值校验（server/client）
- `internal/ir/builder.go`：`buildCreate` 支持 key 字段前置（client 模式）；Create HTTP annotation 路径拼 key 叶子
- `internal/render/template.go`：request message 与 HTTP path 渲染
- `examples/book/api.yaml` + `examples/book/e2e_http_test.go` + 重新生成 `examples/book/generated/`：新增 client-key 实体做端到端验证
- `README.md` / `README_EN.md` / `design-v2.md`

### 兼容性

- 默认行为不变（`server`），现有 api.yaml 与生成代码零影响
- `key: client` 是新模式，无旧模式字段号破坏问题

### 注意事项

- 仓库中存在活跃变更 `testcase-e2e-suite`，但其不修改 `examples/` 代码（仅复制 mock 到 `testcase/`），两者无实际冲突，可并行合入
- my-wechat（/Users/gavinnzhu/temp/wechat/my-wechat）：本能力发版后，其 28 个实体将全部切换为 `create: { key: client }`（在 my-wechat 仓库单独执行，不属于本提案）
