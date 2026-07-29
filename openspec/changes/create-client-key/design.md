## Context

api-gen 的实体级 Create 形态自 apigen-core 起固定为"server 生成主键"：Create 请求内嵌各资源（可选，支持部分创建）、响应只回带 key。design-v2 决策 #27 明确"统一 server 生成，Create 请求不含 key、响应回带 key；无 `generates` 配置"。

my-wechat 项目（微信解密数据分析系统）是首个重度使用方，其场景为"解密数据库导入 + 只读分析"：全部实体使用自然主键，数据由导入脚本写入，服务端无法生成主键。当前契约下导入脚本无法通过 Create 写入数据（请求里连 key 字段都没有）。

## Goals / Non-Goals

**Goals:**
- `create` 增加 key 生成方配置：`server`（默认，现状）/ `client`
- client 模式下 Create 请求携带 key（field 1），资源字段顺延
- client 模式下 HTTP 路径携带 key 叶子段（复用既有递归展开），body 携带资源
- 非法配置 fail-fast 校验
- 默认行为完全不变，向后兼容

**Non-Goals:**
- 不改变 server 模式的任何生成形态
- 不实现服务端主键生成策略（自增/雪花等始终属于服务端实现层）
- 不定义重复 key 创建的服务端行为（实现层返回 ALREADY_EXISTS，不属于代码生成）
- 不为 Delete/Get/Update 增加新模式（它们本就在请求/path 中携带 key）
- my-wechat 侧的 api.yaml 切换不在本提案范围（api-gen 发版后在其仓库执行）

## Decisions

### D1: 配置形态 `create: { key: server|client }`，默认 server

**决策**：`Entity.Create` 从 `*struct{}` 扩展为 `*CreateDef{ Key string }`，`key` 取值 `server` / `client`，空串等价 `server`。

**理由**：
- design-v2 决策 #27 曾预留"谁生成主键"的策略位（当时结论：无配置、统一 server），本提案是把该策略位落地
- 命名用 `key` 而非 `generates`：与 YAML 中既有 `key:`（KeyDef）语义呼应，读作"create 的 key 由谁提供"
- 空值默认 server 保证存量配置零影响

### D2: client 模式请求形态：key=1 前置，资源顺延

**决策**：`Create<Entity>Request { <KeyType> key = 1; <resource_1> = 2; ... }`；响应保持 `{ <KeyType> key = 1 }` 不变。

**理由**：
- key 放 field 1 与 Delete/Get/Update 等各 wrapper 的 key 位置约定一致
- 响应回显 key 保持两种模式响应形态统一，客户端代码无需分支
- 部分创建语义（各资源可选）不受影响

### D3: HTTP 路径携带 key 叶子段，body 携带资源

**决策**：`post: "/{prefix}/{Service}/{collection}/{key叶子段...}"`，`body: "*"`。

**理由**：
- 复用 Delete/Get 已有的 key 类型树递归叶子展开（KeyLeaves / joinHTTPPath），无新机制
- grpc-gateway 语义：path 变量注入请求字段，`body: "*"` 映射其余字段（资源），key 与资源天然分流
- 与 AIP-133 user-specified ID 的 HTTP 习惯一致（POST 到带完整 resource name 的路径）

### D4: 校验 fail-fast

**决策**：`create.key` 取值不在 `{server, client}` 时 validate 报错，错误信息指出实体名与合法取值。

**理由**：拼写错误（如 `key: clinet`）若静默回退 server 会产生难以察觉的契约偏差。

### D5: api-linter 豁免按需补充

**决策**：client 模式 Create 请求带 key 字段可能触发 `core::0133` 系列告警（对请求字段命名/顺序的假设），实施时在 internal/lint 豁免表中按需补充，并记录到 design-v2 §16。

**理由**：豁免应精确到实际触发的规则，不预先过度豁免。

## Risks / Trade-offs

### R1: 两种模式并存增加理解成本
- **风险**：使用者需要理解 server/client 差异
- **缓解**：默认 server 不变；README 用对照表说明两种模式的请求/路径形态；my-wechat 是首个 client 模式用例

### R2: 模式一经发布不可切换
- **风险**：若某实体从 client 切回 server，请求字段号变化（key 消失、资源回到 1 起）是 breaking change
- **缓解**：模式选择属于实体建模决策，文档中明确"一经发布不要切换模式"

### R3: 重复 key 创建的行为未定义
- **风险**：client 指定已存在的 key 时服务端行为（upsert? 报错?）不在生成契约内
- **缓解**：定位为服务端实现层语义（建议 ALREADY_EXISTS），文档注明；不在本提案展开

### R4: 与活跃变更 testcase-e2e-suite 的潜在冲突
- **风险**：两个变更都触碰 `examples/book/e2e_http_test.go`
- **缓解**：本提案只新增独立测试函数，不改既有用例；若 testcase-e2e-suite 先合入，rebase 对齐

## Implementation Notes

### 实施后修正：R4 风险描述不准确
实施时确认 `testcase-e2e-suite` 的 proposal 明确声明"不修改任何 `internal/` 或 `examples/` 代码"——它只是从 `e2e_http_test.go` 复制 mock 到 `testcase/positive/helpers_test.go`。因此两者无实际冲突，可并行合入。

### 实施后确认：api-linter 豁免
D5 中担心的 client 模式新增 api-linter 告警未实际触发新规则。现有 `core::0133::field-numbers`（多资源载荷字段号）和 `core::0133::request-parent-field`（无独立 parent 字段）已覆盖 client 模式的字段号偏离——key=1 前置 + 资源顺延仍属于"非标准字段号布局"。`core::0133::http-body`（body:"*"）也仍然适用。无需新增豁免，design-v2 §16 豁免表无需修改。

### 实施细节
- `CreateIR` 新增 `KeyField *FieldIR`（指针类型，nil 表示 server 模式），避免值类型零值歧义
- `buildCreate` 增加 `createDef *apigenyaml.CreateDef` 参数，通过 `isClient` 标志控制 key 前置与资源起始字段号
- `buildCreateAnnotationWithResources` 增加 `createDef` 参数，client 模式复用 `h.keyLeaves`（与 Delete 的 KeyLeaves 填充方式一致）
- `renderMessages` 在 RequestFields 前检查 `KeyField != nil` 并渲染
- e2e 新增 `note` 实体（`NoteId{string id}` + `NoteMeta{string title; string body}`），验证 POST `/library/LibraryService/note/{key.id}` 路径注入 key + body 注入资源
