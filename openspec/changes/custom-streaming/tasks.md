## 1. YAML 配置扩展（stream 字段解析与校验）

- [x] 1.1 custom_methods[].stream 配置解析与取值校验  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 1.1.1 写失败测试：`internal/yaml/parser_test.go`（解析 `stream: server` / `stream: client` / `stream: bidi` / 省略四种形态）、`internal/yaml/validate_test.go`（非法值如 `stream: both` 报错，错误信息含方法名与合法取值）
  - [x] 1.1.2 验证测试失败（运行：`go test ./internal/yaml/ -count=1`，确认失败原因是 Stream 字段/校验逻辑不存在）
  - [x] 1.1.3 写最小实现：`internal/yaml/parser.go`（`CustomMethod` 新增 `Stream string` 字段）、`internal/yaml/validate.go`（取值校验，空串等价 unary）
  - [x] 1.1.4 验证测试通过（运行：`go test ./internal/yaml/... -count=1`，确认所有测试通过，输出干净）
  - [x] 1.1.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 1.2 HTTP 不兼容校验  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 1.2.1 写失败测试：`internal/yaml/validate_test.go`（`http.enable=true` + `stream: client` 报错；`http.enable=true` + `stream: bidi` 报错；`http.enable=true` + `stream: server` 通过；`http.enable=false` + 任意 stream 通过）
  - [x] 1.2.2 验证测试失败（运行：`go test ./internal/yaml/ -count=1 -run HTTPIncompatible`，确认失败原因是校验逻辑不存在）
  - [x] 1.2.3 写最小实现：`internal/yaml/validate.go`（在 custom_methods 循环中检查 stream + http.enable 组合，client/bidi 报错）
  - [x] 1.2.4 验证测试通过（运行：`go test ./internal/yaml/... -count=1`，确认所有测试通过，输出干净）
  - [x] 1.2.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 1.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/custom-streaming/specs/*.md` 和 `openspec/changes/custom-streaming/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 2. IR 与 proto 渲染（流式 RPC 生成）

- [x] 2.1 CustomMethodIR.Stream 字段与 buildService 透传  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 2.1.1 写失败测试：`internal/ir/builder_test.go`（`stream: server` 时 IR.CustomMethods[0].Stream == "server"；`stream: bidi` 时 == "bidi"；省略时 == ""）
  - [x] 2.1.2 验证测试失败（运行：`go test ./internal/ir/ -count=1 -run CustomMethod.*Stream`，确认失败原因是 Stream 字段不存在）
  - [x] 2.1.3 写最小实现：`internal/ir/builder.go`（`CustomMethodIR` 新增 `Stream string`，`buildService` 透传 `cm.Stream`）
  - [x] 2.1.4 验证测试通过（运行：`go test ./internal/ir/... -count=1`，确认所有测试通过，输出干净）
  - [x] 2.1.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 2.2 renderRPCWithHTTP 支持 stream 标记  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 2.2.1 写失败测试：`internal/render/template_test.go`（unary + 无 HTTP：`rpc X(Req) returns (Resp);`；server + 无 HTTP：`returns (stream Resp)`；server + HTTP：`returns (stream Resp) { option ... }`；client + 无 HTTP：`rpc X(stream Req) returns (Resp);`；bidi + 无 HTTP：`rpc X(stream Req) returns (stream Resp);`）
  - [x] 2.2.2 验证测试失败（运行：`go test ./internal/render/ -count=1 -run Stream`，确认失败原因是 streamMode 参数不存在）
  - [x] 2.2.3 写最小实现：`internal/render/http.go`（`renderRPCWithHTTP` 新增 `streamMode string` 参数，按 D3 规则在 Req/Resp 前插入 `stream`）、`internal/render/template.go`（`renderServiceRPCs` 传 `""`，custom_methods 传 `cm.Stream`）
  - [x] 2.2.4 验证测试通过（运行：`go test ./... -count=1`，确认所有测试通过，输出干净）
  - [x] 2.2.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 2.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/custom-streaming/specs/*.md` 和 `openspec/changes/custom-streaming/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 3. 端到端验证与文档

- [x] 3.1 examples/book 新增流式 custom_method e2e 验证  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 3.1.1 写失败测试：`examples/book/e2e_grpc_test.go`（新增 server-streaming custom_method 的 gRPC 调用测试：客户端发起请求、服务端用 stream.Send 逐条返回、客户端 Recv 收齐验证）。注意：client/bidi 需要流式客户端，至少验证 server-streaming；client-streaming 可选追加
  - [x] 3.1.2 验证测试失败（运行：`cd examples/book && go test ./... -count=1 -run Stream`，确认失败原因是流式方法不存在）
  - [x] 3.1.3 写最小实现：`examples/book/api.yaml`（LibraryService 新增一个 `stream: server` 的 custom_method，如 `StreamBookMetas`；HTTP 保持启用验证 server-stream + HTTP 并存），重新生成 `examples/book/generated/`
  - [x] 3.1.4 验证测试通过（运行：`cd examples/book && go test ./... -count=1`，确认所有测试通过，输出干净）
  - [x] 3.1.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 3.2 更新用户文档  <!-- 非 TDD 任务：使用 3 步子任务 -->
  - [x] 3.2.1 执行变更：
    - `.claude/skills/apigen-cli/SKILL.md`（方法生成映射表补充 stream 列；HTTP 路径表注明 server-stream 支持）
    - `.claude/skills/apigen-cli/references/config-schema.md`（`custom_methods[].stream` 字段说明 + 取值表 + HTTP 兼容矩阵）
    - `.claude/skills/apigen-cli/references/examples.md`（流式 custom_method 示例）
    - `README.md` / `README_EN.md`（custom_methods 配置说明补充 stream）
  - [x] 3.2.2 验证无回归（运行：`go build ./... && go test ./... -count=1`，确认输出干净）
  - [x] 3.2.3 检查：确认文档描述与生成契约一致（stream 取值、proto 语法、HTTP 兼容性），中英文档同步

- [x] 3.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/custom-streaming/specs/*.md` 和 `openspec/changes/custom-streaming/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 4. Documentation Sync (Required)

- [x] 4.1 sync design.md: record technical decisions, deviations, and implementation details after each code change
- [x] 4.2 sync tasks.md: 逐一检查所有顶层任务及其子任务，将已完成但仍为 `[ ]` 的条目标记为 `[x]`；每次更新只修改 `[ ]` → `[x]`，禁止修改任何任务描述文字
- [x] 4.3 sync proposal.md: update scope/impact if changed
- [x] 4.4 sync specs/*.md: update requirements if changed
- [x] 4.5 Final review: ensure all OpenSpec docs reflect actual implementation
