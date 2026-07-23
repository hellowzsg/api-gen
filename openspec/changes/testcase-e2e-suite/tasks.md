## 1. testcase 模块骨架与正向 fixtures <!-- 轻量任务组：跳过独立审查，变更纳入后续任务组统一审查 -->

- [x] 1.1 创建 testcase 目录结构与 go.mod <!-- 非 TDD 任务 -->
  - [x] 1.1.1 执行变更：创建 `testcase/go.mod`（module github.com/hellowzsg/api-gen/testcase）、`testcase/README.md`、`testcase/.gitignore`
  - [x] 1.1.2 验证无回归（运行：`cd testcase && go mod tidy`，确认依赖解析正常）
  - [x] 1.1.3 检查：确认目录结构完整（fixtures/、positive/、negative/ 均已创建）

- [x] 1.2 创建 book fixture（完整 HTTP+OpenAPI） <!-- 非 TDD 任务 -->
  - [x] 1.2.1 执行变更：复制 `examples/book/api.yaml` → `testcase/fixtures/book/api.yaml`（调整 go_repo 为 testcase 路径），复制 `examples/book/proto/` → `testcase/fixtures/book/proto/`
  - [x] 1.2.2 验证无回归（运行：`go build -o /tmp/apigen ./cmd/apigen && /tmp/apigen generate -f testcase/fixtures/book/api.yaml`，确认生成产物正常）
  - [x] 1.2.3 检查：确认 fixture 包含 api.yaml + proto/，生成产物目录结构正确

- [x] 1.3 创建 simple fixture（P0 纯 gRPC） <!-- 非 TDD 任务 -->
  - [x] 1.3.1 执行变更：复制 `examples/simple/api.yaml` → `testcase/fixtures/simple/api.yaml`，复制对应 proto/
  - [x] 1.3.2 验证无回归（运行：`/tmp/apigen generate -f testcase/fixtures/simple/api.yaml`，确认生成产物正常）
  - [x] 1.3.3 检查：确认 simple fixture 无 HTTP 配置（P0 纯 gRPC）

- [x] 1.4 创建 edge fixture（边界条件） <!-- 非 TDD 任务 -->
  - [x] 1.4.1 执行变更：创建 `testcase/fixtures/edge/api.yaml`（多 entity、多 resource、嵌套 key、custom_method）和 `testcase/fixtures/edge/proto/` 对应 proto 定义
  - [x] 1.4.2 验证无回归（运行：`/tmp/apigen generate -f testcase/fixtures/edge/api.yaml`，确认生成产物正常）
  - [x] 1.4.3 检查：确认 edge fixture 覆盖嵌套 key、多资源、custom_method 边界场景

（无独立代码审查任务 — 变更纳入后续常规任务组的审查范围）

## 2. 正向 e2e 测试套件

- [x] 2.1 创建测试 helper 与 mock 基础设施 <!-- TDD 任务 -->
  - [x] 2.1.1 写失败测试：`testcase/positive/helpers_test.go` — 编写 newGRPCServer、newGatewayMux、doReq、mustReadJSON helper 函数的最小测试用例
  - [x] 2.1.2 验证测试失败（运行：`cd testcase && go test ./positive/... -run TestHelper -v`，确认失败原因是 helper 未实现）
  - [x] 2.1.3 写最小实现：`testcase/positive/helpers_test.go` — 实现 mock server（mockLibraryServer、mockAdminServer）和 helper 函数，从 examples/book/e2e_http_test.go 复制并适配
  - [x] 2.1.4 验证测试通过（运行：`cd testcase && go test ./positive/... -run TestHelper -v`，确认所有测试通过，输出干净）
  - [x] 2.1.5 重构：提取通用 mock 基础设施，消除与 examples/book 的重复代码

- [x] 2.2 generate 产物验证测试 <!-- TDD 任务 -->
  - [x] 2.2.1 写失败测试：`testcase/positive/generate_test.go` — 编写测试验证 book fixture 生成的 proto 文件结构（service、RPC 方法、message 字段、AdminService 收窄）
  - [x] 2.2.2 验证测试失败（运行：`cd testcase && go test ./positive/... -run TestGenerate -v`，确认失败原因是测试文件未实现）
  - [x] 2.2.3 写最小实现：`testcase/positive/generate_test.go` — 实现读取生成 proto 文件并断言内容的测试
  - [x] 2.2.4 验证测试通过（运行：`cd testcase && go test ./positive/... -run TestGenerate -v`，确认所有测试通过）
  - [x] 2.2.5 重构：提取 proto 文件读取和断言 helper

- [x] 2.3 gRPC 全方法正向测试 <!-- TDD 任务 -->
  - [x] 2.3.1 写失败测试：`testcase/positive/grpc_test.go` — 编写 LibraryService 全 9 个 RPC + AdminService 收窄方法 + Unimplemented 验证的测试
  - [x] 2.3.2 验证测试失败（运行：`cd testcase && go test ./positive/... -run TestGRPC -v`，确认失败）
  - [x] 2.3.3 写最小实现：`testcase/positive/grpc_test.go` — 实现 gRPC 全方法测试，覆盖字段穿透、version、FieldMask、Filter、分页、批量
  - [x] 2.3.4 验证测试通过（运行：`cd testcase && go test ./positive/... -run TestGRPC -v`，确认所有测试通过）
  - [x] 2.3.5 重构：提取 gRPC 测试通用模式

- [x] 2.4 HTTP gateway 全路由正向测试 <!-- TDD 任务 -->
  - [x] 2.4.1 写失败测试：`testcase/positive/http_test.go` — 编写所有 HTTP 端点（Create POST、Delete DELETE、Get GET、List POST、BatchGet POST、Update PATCH、custom method 冒号语法）的测试
  - [x] 2.4.2 验证测试失败（运行：`cd testcase && go test ./positive/... -run TestHTTP -v`，确认失败）
  - [x] 2.4.3 写最小实现：`testcase/positive/http_test.go` — 实现 HTTP 全路由测试，覆盖 protojson 序列化、路径段绑定、body 解码
  - [x] 2.4.4 验证测试通过（运行：`cd testcase && go test ./positive/... -run TestHTTP -v`，确认所有测试通过）
  - [x] 2.4.5 重构：提取 HTTP 请求构造和响应断言 helper

- [x] 2.5 OpenAPI spec 正向测试 <!-- TDD 任务 -->
  - [x] 2.5.1 写失败测试：`testcase/positive/openapi_test.go` — 编写 swagger.json 有效性、paths 完整性、HTTP verb 正确性测试
  - [x] 2.5.2 验证测试失败（运行：`cd testcase && go test ./positive/... -run TestOpenAPI -v`，确认失败）
  - [x] 2.5.3 写最小实现：`testcase/positive/openapi_test.go` — 实现 OpenAPI spec 验证测试
  - [x] 2.5.4 验证测试通过（运行：`cd testcase && go test ./positive/... -run TestOpenAPI -v`，确认所有测试通过）
  - [x] 2.5.5 重构：消除与 examples/book/e2e_openapi_test.go 的重复

- [x] 2.6 simple fixture P0 纯 gRPC 验证 <!-- TDD 任务 -->
  - [x] 2.6.1 写失败测试：`testcase/positive/simple_grpc_test.go` — 编写测试验证 simple fixture 生成的 proto 不含 google.api.http 注解、不含 google/api/annotations.proto import、仅 gRPC service 定义
  - [x] 2.6.2 验证测试失败（运行：`cd testcase && go test ./positive/... -run TestSimpleGRPC -v`，确认失败）
  - [x] 2.6.3 写最小实现：`testcase/positive/simple_grpc_test.go` — 实现 P0 纯 gRPC 模式验证
  - [x] 2.6.4 验证测试通过（运行：`cd testcase && go test ./positive/... -run TestSimpleGRPC -v`，确认所有测试通过）
  - [x] 2.6.5 重构：提取 proto 内容断言 helper 复用

- [x] 2.7 edge fixture 边界条件验证 <!-- TDD 任务 -->
  - [x] 2.7.1 写失败测试：`testcase/positive/edge_test.go` — 编写测试验证多 entity 各自 RPC 生成、嵌套 key 路径段绑定、version.kind=WEAK 生成 UInt64Value wrapper、version.kind=NONE 无 version 字段
  - [x] 2.7.2 验证测试失败（运行：`cd testcase && go test ./positive/... -run TestEdge -v`，确认失败）
  - [x] 2.7.3 写最小实现：`testcase/positive/edge_test.go` — 实现 edge fixture 边界条件验证
  - [x] 2.7.4 验证测试通过（运行：`cd testcase && go test ./positive/... -run TestEdge -v`，确认所有测试通过）
  - [x] 2.7.5 重构：提取 version 字段断言 helper

- [x] 2.9 edge fixture 跨 package 类型引用验证 <!-- TDD 任务（补充场景） -->
  - [x] 2.9.1 执行变更：将 edge fixture 的 tag entity 类型（TagId、Tag）从 `edge.example` package 拆分到新建的 `edge.tag` package（`proto/edge/tag/tag.proto`）；更新 `api.yaml` 中 tag entity 的类型引用为全限定名 `edge.tag.TagId` / `edge.tag.Tag`
  - [x] 2.9.2 验证生成正确（运行 `generate` + `build`，确认跨 package 引用正常）
  - [x] 2.9.3 写测试：`testcase/positive/edge_test.go` — `TestEdge_CrossPackageTypes`，验证 doc entity 使用 `edge.example.*` 类型、tag entity 使用 `edge.tag.*` 类型、两个 package 的 import 都存在、类型不交叉污染
  - [x] 2.9.4 验证测试通过（运行：`go test ./positive/... -run TestEdge -v`）
  - [x] 2.9.5 更新 design.md 记录 D11 决策

- [x] 2.8 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/testcase-e2e-suite/specs/*.md` 和 `openspec/changes/testcase-e2e-suite/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认
  - subagent 模式下：本任务仍须执行（子代理内置审查为内部质量门控，不替代本任务的用户可见审查）

## 3. 反向 fixtures 创建 <!-- 轻量任务组：跳过独立审查，变更纳入后续任务组统一审查 -->

- [x] 3.1 创建 YAML 解析层错误 fixtures <!-- 非 TDD 任务 -->
  - [x] 3.1.1 执行变更：在 `testcase/fixtures/invalid/` 下创建 missing_syntax.yaml、missing_name.yaml、invalid_name_format.yaml、missing_entities.yaml、entity_missing_name.yaml、entity_missing_key_type.yaml、entity_no_resources.yaml、resource_missing_name.yaml、resource_missing_type.yaml、resource_missing_version.yaml、unknown_yaml_field.yaml
  - [x] 3.1.2 验证无回归（运行：`ls testcase/fixtures/invalid/`，确认 11 个文件存在）
  - [x] 3.1.3 检查：每个文件内容极简，仅触发对应错误，注释标明错误码

- [x] 3.2 创建类型引用与服务引用错误 fixtures <!-- 非 TDD 任务 -->
  - [x] 3.2.1 执行变更：创建 key_type_empty.yaml、key_type_dot_prefix.yaml、key_type_digit_prefix.yaml、key_type_illegal_char.yaml、filter_type_invalid.yaml、duplicate_entity.yaml、service_ref_undefined_entity.yaml、service_ref_undefined_resource.yaml
  - [x] 3.2.2 验证无回归（运行：`ls testcase/fixtures/invalid/`，确认 8 个新文件存在）
  - [x] 3.2.3 检查：确认每个 fixture 触发唯一错误路径

- [x] 3.3 创建 HTTP 配置与路径变量错误 fixtures <!-- 非 TDD 任务 -->
  - [x] 3.3.1 执行变更：创建 http_without_googleapis.yaml、invalid_body_style.yaml、empty_path_var.yaml、leading_dot_path_var.yaml、trailing_dot_path_var.yaml、unreachable_path_var.yaml
  - [x] 3.3.2 验证无回归（运行：`ls testcase/fixtures/invalid/`，确认 6 个新文件存在）
  - [x] 3.3.3 检查：确认 HTTP 相关错误覆盖完整

- [x] 3.4 创建插件、IR、key leaves、option 错误 fixtures <!-- 非 TDD 任务 -->
  - [x] 3.4.1 执行变更：创建 unknown_js_plugin.yaml、http_key_descriptor_missing.yaml、resource_style_multi_create.yaml、key_with_repeated.yaml、key_with_map.yaml、key_with_oneof.yaml、circular_key_ref.yaml、option_invalid_target.yaml、option_field_no_path.yaml、option_message_no_path.yaml、option_rpc_no_path.yaml、option_empty_name.yaml、option_name_with_space.yaml、option_name_illegal_char.yaml、key_type_not_in_proto.yaml、resource_type_not_in_proto.yaml
  - [x] 3.4.2 验证无回归（运行：`ls testcase/fixtures/invalid/`，确认 16 个新文件存在）
  - [x] 3.4.3 检查：确认所有 13 层错误路径均有对应 fixture

- [x] 3.5 创建依赖解析层错误 fixtures <!-- 非 TDD 任务 -->
  - [x] 3.5.1 执行变更：创建 `testcase/fixtures/invalid/glob_no_match.yaml`（import_protos.path 指向空目录）、`testcase/fixtures/invalid/protocompile_error/`（含语法错误的 proto 文件）、`testcase/fixtures/invalid/corrupt_api_lock.lock`（损坏的 api.lock 文件）
  - [x] 3.5.2 验证无回归（运行：`ls testcase/fixtures/invalid/`，确认 3 个新 fixture 存在）
  - [x] 3.5.3 检查：确认依赖解析层 3 个错误路径（protocompile 失败、api.lock 损坏、glob 无匹配）均有对应 fixture

（无独立代码审查任务 — 变更纳入后续常规任务组的审查范围）

## 4. 反向 e2e 测试套件

- [x] 4.1 generate 错误测试（YAML 解析 + 类型引用 + Service 引用 + HTTP 配置 + 路径语法 + 插件） <!-- TDD 任务 -->
  - [x] 4.1.1 写失败测试：`testcase/negative/generate_errors_test.go` — 编写对 fixtures/invalid/ 下 YAML 解析层（A1-A12）、类型引用（B1-B6）、Service 引用（C1-C3）、HTTP 配置（D1-D2）、路径语法（E1-E3）、插件（F1）错误 fixture 的测试
  - [x] 4.1.2 验证测试失败（运行：`cd testcase && go test ./negative/... -run TestGenerateError -v`，确认失败）
  - [x] 4.1.3 写最小实现：`testcase/negative/generate_errors_test.go` — 实现 table-driven 测试，对每个 invalid fixture 运行 generate 并断言错误消息
  - [x] 4.1.4 验证测试通过（运行：`cd testcase && go test ./negative/... -run TestGenerateError -v`，确认所有测试通过）
  - [x] 4.1.5 重构：提取错误断言 helper，统一 table-driven 结构

- [x] 4.2 IR 构建错误测试（key leaves + option + 路径可达性 + CLI 生成层） <!-- TDD 任务 -->
  - [x] 4.2.1 写失败测试：`testcase/negative/ir_errors_test.go` — 编写 IR 构建层（G1-G2）、key leaves（G3-G6）、路径可达性（H1）、option（I1-I7）、CLI 生成层（J1-J2）错误 fixture 的测试
  - [x] 4.2.2 验证测试失败（运行：`cd testcase && go test ./negative/... -run TestIRError -v`，确认失败）
  - [x] 4.2.3 写最小实现：`testcase/negative/ir_errors_test.go` — 实现对 IR 层错误的测试，部分需通过 apigen build（而非 generate）触发
  - [x] 4.2.4 验证测试通过（运行：`cd testcase && go test ./negative/... -run TestIRError -v`，确认所有测试通过）
  - [x] 4.2.5 重构：提取 build 错误测试通用模式

- [x] 4.3 HTTP 运行时反向测试 <!-- TDD 任务 -->
  - [x] 4.3.1 写失败测试：`testcase/negative/http_negative_test.go` — 编写 404 未注册路由、错误 HTTP method、无效 JSON body、AdminService 收窄路由 404 的测试
  - [x] 4.3.2 验证测试失败（运行：`cd testcase && go test ./negative/... -run TestHTTPNegative -v`，确认失败）
  - [x] 4.3.3 写最小实现：`testcase/negative/http_negative_test.go` — 实现 HTTP 运行时反向路径测试
  - [x] 4.3.4 验证测试通过（运行：`cd testcase && go test ./negative/... -run TestHTTPNegative -v`，确认所有测试通过）
  - [x] 4.3.5 重构：提取 HTTP 反向测试通用模式

- [x] 4.4 gRPC 运行时反向测试 <!-- TDD 任务 -->
  - [x] 4.4.1 写失败测试：`testcase/negative/grpc_negative_test.go` — 编写未注册方法 Unimplemented、nil 请求、context 取消的测试
  - [x] 4.4.2 验证测试失败（运行：`cd testcase && go test ./negative/... -run TestGRPCNegative -v`，确认失败）
  - [x] 4.4.3 写最小实现：`testcase/negative/grpc_negative_test.go` — 实现 gRPC 运行时反向路径测试
  - [x] 4.4.4 验证测试通过（运行：`cd testcase && go test ./negative/... -run TestGRPCNegative -v`，确认所有测试通过）
  - [x] 4.4.5 重构：提取 gRPC 反向测试通用模式

- [x] 4.5 依赖解析层错误测试 <!-- TDD 任务 -->
  - [x] 4.5.1 写失败测试：`testcase/negative/dep_errors_test.go` — 编写 glob 无匹配文件（`no .proto files matched pattern`）、protocompile 编译失败（`protocompile failed`）、api.lock 损坏（非 ErrNotExist 的读取错误）的测试
  - [x] 4.5.2 验证测试失败（运行：`cd testcase && go test ./negative/... -run TestDepError -v`，确认失败）
  - [x] 4.5.3 写最小实现：`testcase/negative/dep_errors_test.go` — 实现依赖解析层错误测试，对 3.5 创建的 fixtures 运行 generate/build 并断言错误消息
  - [x] 4.5.4 验证测试通过（运行：`cd testcase && go test ./negative/... -run TestDepError -v`，确认所有测试通过）
  - [x] 4.5.5 重构：提取依赖错误测试通用模式

- [x] 4.6 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/testcase-e2e-suite/specs/*.md` 和 `openspec/changes/testcase-e2e-suite/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件（含任务组 3 轻量变更）
    - `{BASE_SHA}` → 任务组 3 开始前的 commit SHA（轻量组合并审查）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认
  - subagent 模式下：本任务仍须执行（子代理内置审查为内部质量门控，不替代本任务的用户可见审查）

## 5. CI 集成与文档完善

- [x] 5.1 新增 CI testcase-e2e job <!-- 非 TDD 任务 -->
  - [x] 5.1.1 执行变更：`.github/workflows/ci.yml` 新增 `testcase-e2e` job，包含 checkout、Go setup、插件安装、apigen build、fixtures 生成、正向/反向测试运行
  - [x] 5.1.2 验证无回归（运行：`cat .github/workflows/ci.yml | grep testcase-e2e`，确认 job 已添加）
  - [x] 5.1.3 检查：确认 CI job 步骤完整（生成 fixtures → 运行 positive → 运行 negative）

- [x] 5.2 完善 testcase/README.md <!-- 非 TDD 任务 -->
  - [x] 5.2.1 执行变更：`testcase/README.md` 编写测试体系说明（目录结构、运行方式、新增 fixture 指南、正反向测试编写规范）
  - [x] 5.2.2 验证无回归（运行：`cat testcase/README.md`，确认内容完整）
  - [x] 5.2.3 检查：确认 README 涵盖所有必要信息

- [x] 5.3 补充 .gitignore <!-- 非 TDD 任务 -->
  - [x] 5.3.1 执行变更：项目根 `.gitignore` 添加 `testcase/fixtures/*/generated/`
  - [x] 5.3.2 验证无回归（运行：`git status testcase/fixtures/book/generated/`，确认被忽略）
  - [x] 5.3.3 检查：确认所有 fixture 的生成产物目录均被忽略

- [x] 5.4 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/testcase-e2e-suite/specs/*.md` 和 `openspec/changes/testcase-e2e-suite/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认
  - subagent 模式下：本任务仍须执行（子代理内置审查为内部质量门控，不替代本任务的用户可见审查）

## 6. PreCI 代码规范检查（skip_preci: true — 跳过）

- [x] 6.1 PreCI 检查已跳过（项目配置 skip_preci: true，aip-gen 项目不执行 PreCI 代码规范检查）

## 7. Documentation Sync (Required)

- [x] 7.1 sync design.md: record technical decisions, deviations, and implementation details after each code change
- [x] 7.2 sync tasks.md: 逐一检查所有顶层任务及其子任务，将已完成但仍为 `[ ]` 的条目标记为 `[x]`；每次更新只修改 `[ ]` → `[x]`，禁止修改任何任务描述文字
- [x] 7.3 sync proposal.md: update scope/impact if changed
- [x] 7.4 sync specs/*.md: update requirements if changed
- [x] 7.5 Final review: ensure all OpenSpec docs reflect actual implementation
