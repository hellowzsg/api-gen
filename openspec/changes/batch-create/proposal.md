## Why

api-gen 当前支持单条 Create（实体级，`create: {}` 或 `create: { key: client }`）和批量查询 BatchGet（资源级，`reader.batch: true`），但缺少批量创建（BatchCreate）能力。

首个重度使用方 my-wechat 项目的导入场景需要批量写入：解密后的数据按批次导入数据库，逐条 Create 的网络往返成本不可接受。AIP-233 定义了批量创建的标准模式，api-gen 需要补齐这一能力。

## What Changes

### 1. `create.batch` 配置扩展（向后兼容）

- `create: {}` / `create: { key: server }`：现状，仅生成单条 `Create`
- `create: { key: client }`：现状，仅生成单条 client-key `Create`
- `create: { batch: true }`：新增，额外生成 `BatchCreate` RPC（server-key 模式）
- `create: { key: client, batch: true }`：新增，额外生成 `BatchCreate` RPC（client-key 模式，每个 request 携带各自的 key）
- `batch: true` 可与 `key: server` / `key: client` 任意组合

### 2. BatchCreate 生成契约

- **RPC 名**：`BatchCreate<Entity>s`（复数，与 BatchGet 命名一致）
- **Request**：`{ repeated Create<Entity>Request requests = 1 }`（复用单条 Create 的 Request 形态作为元素）
- **Response**：`{ repeated <KeyType> keys = 1 }`（返回创建成功的 keys）
- **粒度**：实体级（与 Create 同级），不在资源级生成

### 3. HTTP 路由

- `POST /{prefix}/{Service}/{collection}/batchCreate body:"*"`
- 与 BatchGet 的 `/{resource}/batchGet`、List 的 `/{resource}/list` 对称
- 不携带 key 路径段（集合级操作，key 在 body 的 requests 内）
- 即使 client-key 模式也不在 path 携带 key（批量操作的 keys 在 body 内）

### 4. Service 级 narrowing

- BatchCreate 随 Create 一起暴露/收窄
- Create 被 narrowing 掉时 BatchCreate 也一起消失（不存在单独收窄 BatchCreate 的场景）
- service 级 `resources` narrowing 不影响 BatchCreate（BatchCreate 是实体级，不依赖资源级 narrowing）

### 5. api-linter 豁免

预期触发 `core::0233` 系列（BatchCreate 标准 AIP），需在豁免表中补充：
- `core::0233::request-message-name`（BatchCreate wrapper 命名）
- `core::0233::http-body`（POST + `body:"*"`）
- `core::0233::http-method`（POST）

### 6. 测试用例

- `testcase/fixtures/book/api.yaml`：新增 `batch: true` 的实体，重新生成 fixtures
- `testcase/positive/grpc_test.go` + `http_test.go`：新增 BatchCreate gRPC / HTTP e2e 测试
- `testcase/negative/`：新增 BatchCreate 相关的负面测试（如 `batch_without_create` 非法配置）
- `examples/book/api.yaml`：新增 `batch: true` 的实体做端到端验证

### 7. 文档

- `README.md` / `README_EN.md`：`create` 配置说明补充 `batch` 字段
- `.claude/skills/apigen-cli/SKILL.md`：方法生成映射表补充 BatchCreate 行
- `.claude/skills/apigen-cli/references/config-schema.md`：`create.batch` 字段说明
- `.claude/skills/apigen-cli/references/examples.md`：BatchCreate 示例
- `design-v2.md`：决策表更新（BatchCreate 实体级方法）

## Impact

### 受影响的代码

- `internal/yaml/parser.go`：`CreateDef` 新增 `Batch bool` 字段
- `internal/yaml/validate.go`：无额外校验（`batch` 在 `create` 内，天然要求 `create` 存在；bool 字段无需取值校验）
- `internal/ir/builder.go`：新增 `BatchCreateIR`，`EntityIR.BatchCreate *BatchCreateIR`，`buildBatchCreate` 函数，HTTP annotation 生成
- `internal/render/template.go`：渲染 BatchCreate RPC + message
- `internal/render/http.go`：BatchCreate HTTP annotation（`/{collection}/batchCreate`）
- `internal/lint/`：豁免表补充 `core::0233` 系列
- `testcase/fixtures/book/api.yaml` + 重新生成 `testcase/fixtures/book/generated/`
- `testcase/positive/grpc_test.go` + `http_test.go`：BatchCreate e2e 测试
- `testcase/fixtures/invalid/batch_without_create.yaml` + `testcase/negative/generate_errors_test.go`：负面测试
- `examples/book/api.yaml` + 重新生成 `examples/book/generated/` + e2e 测试
- `README.md` / `README_EN.md` / `.claude/skills/apigen-cli/` 文档 / `design-v2.md`

### 兼容性

- `batch` 默认 false，现有配置零影响
- 新增 RPC 不改变既有方法字段号
- `batch` 字段一经发布不可移除（移除属于 breaking change）

### 注意事项

- 仓库中存在活跃变更 `create-client-key`（已实施完毕）、`custom-streaming`（已实施完毕），与本提案无冲突
- BatchCreate 的 `requests` 元素复用 `Create<Entity>Request`，client-key 模式下每个元素携带各自的 key
- 部分创建语义不变：每个 `Create<Entity>Request` 内的各资源字段依然可选
