## 1. YAML 配置扩展（create.batch 解析）

- [x] 1.1 create.batch 配置解析  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 1.1.1 写失败测试：`internal/yaml/parser_test.go`（解析 `create: { batch: true }` / `create: { key: client, batch: true }` / `create: { batch: false }` / `create: {}` 四种形态，验证 `CreateDef.Batch` 字段值正确）
  - [x] 1.1.2 验证测试失败（运行：`go test ./internal/yaml/ -count=1 -run CreateDef`，确认失败原因是 `Batch` 字段不存在）
  - [x] 1.1.3 写最小实现：`internal/yaml/parser.go`（`CreateDef` 新增 `Batch bool` 字段，yaml tag `batch,omitempty`）
  - [x] 1.1.4 验证测试通过（运行：`go test ./internal/yaml/... -count=1`，确认所有测试通过，输出干净）
  - [x] 1.1.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 1.2 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/batch-create/specs/*.md` 和 `openspec/changes/batch-create/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 2. BatchCreate IR 与代码生成（IR + 渲染 + HTTP annotation）

- [x] 2.1 BatchCreateIR 与 buildBatchCreate 实现  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 2.1.1 写失败测试：`internal/ir/builder_test.go`（`create: { batch: true }` 时 `EntityIR.BatchCreate` 非 nil，RPCName=`BatchCreateBooks`，RequestName=`BatchCreateBooksRequest`，ResponseName=`BatchCreateBooksResponse`，RequestsField 为 `repeated CreateBookRequest requests=1`，KeysField 为 `repeated BookId keys=1`；省略 batch 时 BatchCreate 为 nil；client-key 模式下 RequestsField.Type 仍为 `Create<Entity>Request`）
  - [x] 2.1.2 验证测试失败（运行：`go test ./internal/ir/ -count=1 -run BatchCreate`，确认失败原因是 `BatchCreateIR`/`buildBatchCreate` 不存在）
  - [x] 2.1.3 写最小实现：`internal/ir/builder.go`（新增 `BatchCreateIR` struct，`EntityIR.BatchCreate *BatchCreateIR` 字段，`buildBatchCreate` 函数生成 IR，`buildEntity` 中 `createDef.Batch` 时调用），HTTP annotation（`buildBatchCreateAnnotation`：`POST /{collection}/batchCreate body:"*"`，无 key leaves）
  - [x] 2.1.4 验证测试通过（运行：`go test ./internal/ir/... -count=1`，确认所有测试通过，输出干净）
  - [x] 2.1.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 2.2 proto 渲染支持 BatchCreate  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 2.2.1 写失败测试：`internal/render/template_test.go`（BatchCreate RPC 渲染为 `rpc BatchCreateBooks(BatchCreateBooksRequest) returns (BatchCreateBooksResponse);`；message 渲染含 `repeated CreateBookRequest requests = 1` 和 `repeated BookId keys = 1`；HTTP 启用时 annotation 为 `post: "/.../book/batchCreate" body: "*"`；省略 batch 时不渲染 BatchCreate RPC/message）
  - [x] 2.2.2 验证测试失败（运行：`go test ./internal/render/ -count=1 -run BatchCreate`，确认失败原因是渲染逻辑不存在）
  - [x] 2.2.3 写最小实现：`internal/render/template.go`（`renderServiceRPCs` 中 BatchCreate 非空时渲染 RPC；`renderMessages` 中渲染 BatchCreate Request/Response message；`narrowEntity` 传递 BatchCreate；`generateExemptions` 补充 `core::0233` 豁免）
  - [x] 2.2.4 验证测试通过（运行：`go test ./... -count=1`，确认所有测试通过，输出干净）
  - [x] 2.2.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 2.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/batch-create/specs/*.md` 和 `openspec/changes/batch-create/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 3. examples/book 端到端验证

- [x] 3.1 examples/book 新增 BatchCreate e2e 验证  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 3.1.1 写失败测试：`examples/book/e2e_grpc_test.go`（新增 `BatchCreateBooks` gRPC 调用测试：发送 2 个 `CreateBookRequest` 元素、服务端验证收到 2 个 requests、响应含 2 个 keys）、`examples/book/e2e_http_test.go`（新增 `POST /library/LibraryService/book/batchCreate` HTTP 测试：body 含 `requests` 数组、验证服务端收到、响应含 `keys` 数组）
  - [x] 3.1.2 验证测试失败（运行：`cd examples/book && go test ./... -count=1 -run BatchCreate`，确认失败原因是 RPC/路由不存在）
  - [x] 3.1.3 写最小实现：`examples/book/api.yaml`（`book` 实体的 `create` 改为 `create: { batch: true }`），重新生成 `examples/book/generated/`
  - [x] 3.1.4 验证测试通过（运行：`cd examples/book && go test ./... -count=1`，确认所有测试通过，输出干净）
  - [x] 3.1.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 3.2 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/batch-create/specs/*.md` 和 `openspec/changes/batch-create/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 4. testcase 测试用例

- [x] 4.1 testcase/fixtures/book 新增 BatchCreate fixture 并重新生成  <!-- 非 TDD 任务：使用 3 步子任务 -->
  - [x] 4.1.1 执行变更：`testcase/fixtures/book/api.yaml`（`book` 实体的 `create` 改为 `create: { batch: true }`），重新生成 `testcase/fixtures/book/generated/`
  - [x] 4.1.2 验证无回归（运行：`cd testcase && go build ./... && go test ./positive/ -count=1`，确认输出干净）
  - [x] 4.1.3 检查：确认生成的 proto 含 `BatchCreateBooks` RPC 与对应 message，HTTP annotation 含 `batchCreate` 路径

- [x] 4.2 testcase/positive 新增 BatchCreate gRPC + HTTP e2e 测试  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 4.2.1 写失败测试：`testcase/positive/grpc_test.go`（新增 `BatchCreateBooks` 子测试：发送 2 个 requests、验证服务端收到、响应含 2 个 keys）、`testcase/positive/http_test.go`（新增 `POST /library/LibraryService/book/batchCreate` 子测试：body 含 `requests` 数组、验证服务端收到、响应含 `keys` 数组）、`testcase/positive/helpers_test.go`（mock server 新增 `lastBatchCreateReq` 字段与 `BatchCreateBooks` 方法实现）
  - [x] 4.2.2 验证测试失败（运行：`cd testcase && go test ./positive/ -count=1 -run BatchCreate`，确认失败原因是方法不存在）
  - [x] 4.2.3 写最小实现：测试文件中的 mock server 补全 `BatchCreateBooks` 方法实现（记录 lastBatchCreateReq、返回 2 个 keys）
  - [x] 4.2.4 验证测试通过（运行：`cd testcase && go test ./positive/ -count=1`，确认所有测试通过，输出干净）
  - [x] 4.2.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 4.3 testcase/negative 新增 BatchCreate 负面测试  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 4.3.1 写失败测试：`testcase/negative/generate_errors_test.go`（新增 `batch_outside_create` 测试用例：`batch: true` 声明在实体级而非 `create` 内部，触发 YAML unknown field 报错）、新增 `testcase/fixtures/invalid/batch_outside_create.yaml`
  - [x] 4.3.2 验证测试失败（运行：`cd testcase && go test ./negative/ -count=1 -run BatchCreate`，确认失败原因是测试用例/fixture 不存在）
  - [x] 4.3.3 写最小实现：`testcase/fixtures/invalid/batch_outside_create.yaml`（实体级 `batch: true`，触发 YAML unknown field 报错）、`testcase/negative/generate_errors_test.go` 新增测试条目
  - [x] 4.3.4 验证测试通过（运行：`cd testcase && go test ./negative/ -count=1`，确认所有测试通过，输出干净）
  - [x] 4.3.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 4.4 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/batch-create/specs/*.md` 和 `openspec/changes/batch-create/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 5. 文档更新

- [x] 5.1 更新用户文档与 skill 文档  <!-- 非 TDD 任务：使用 3 步子任务 -->
  - [x] 5.1.1 执行变更：
    - `README.md`、`README_EN.md`（`create` 配置说明补充 `batch` 字段；方法生成映射表补充 BatchCreate 行；HTTP 路由表补充 `batchCreate` 路径）
    - `.claude/skills/apigen-cli/SKILL.md`（方法生成映射表补充 BatchCreate 行）
    - `.claude/skills/apigen-cli/references/config-schema.md`（`create.batch` 字段说明 + 取值表）
    - `.claude/skills/apigen-cli/references/examples.md`（BatchCreate 示例）
    - `design-v2.md`（决策表更新：BatchCreate 实体级方法；§7.2 配置结构补充 `batch`；§7.5 方法生成表补充 BatchCreate 行；HTTP 路径表补充 `batchCreate`；§16 api-linter 豁免表补充 `core::0233` 系列）
  - [x] 5.1.2 验证无回归（运行：`go build ./... && go test ./... -count=1`，确认输出干净）
  - [x] 5.1.3 检查：确认文档描述与生成契约一致（配置字段、RPC 命名、Request/Response 形态、HTTP 路径），中英文档同步

- [x] 5.2 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/batch-create/specs/*.md` 和 `openspec/changes/batch-create/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 6. Documentation Sync (Required)

- [x] 6.1 sync design.md: record technical decisions, deviations, and implementation details after each code change
- [x] 6.2 sync tasks.md: 逐一检查所有顶层任务及其子任务，将已完成但仍为 `[ ]` 的条目标记为 `[x]`；每次更新只修改 `[ ]` → `[x]`，禁止修改任何任务描述文字
- [x] 6.3 sync proposal.md: update scope/impact if changed
- [x] 6.4 sync specs/*.md: update requirements if changed
- [x] 6.5 Final review: ensure all OpenSpec docs reflect actual implementation
