# List filter_type — 任务清单

## 1. YAML 解析与校验扩展

- [x] 1.1 ListConfig 新增 FilterType 字段  <!-- TDD 任务 -->
  - [x] 1.1.1 写失败测试：`internal/yaml/parser_test.go` — 解析含 `list_config.filter_type: BookMetaFilter` 的 api.yaml，断言 `ListConfig.FilterType == "BookMetaFilter"`
  - [x] 1.1.2 验证测试失败（运行：`go test ./internal/yaml/ -run TestParseFilterType -v -count=1`，确认失败原因是 ListConfig 无 FilterType 字段）
  - [x] 1.1.3 写最小实现：`internal/yaml/parser.go` — `ListConfig` 新增 `FilterType string` 字段，yaml tag `filter_type,omitempty`
  - [x] 1.1.4 验证测试通过（运行：`go test ./internal/yaml/ -run TestParseFilterType -v -count=1`，确认通过）
  - [x] 1.1.5 重构：确认 yaml tag 一致性

- [x] 1.2 validateTypeReferences 扩展 filter_type 语法校验  <!-- TDD 任务 -->
  - [x] 1.2.1 写失败测试：`internal/yaml/validate_test.go` — `filter_type` 为空字符串或 `.Foo`（以点开头）时，断言 ValidateReferences 返回错误
  - [x] 1.2.2 验证测试失败（运行：`go test ./internal/yaml/ -run TestValidateFilterType -v -count=1`，确认失败原因是无校验逻辑）
  - [x] 1.2.3 写最小实现：`internal/yaml/validate.go` — `validateTypeReferences` 中遍历 entities → resources → reader.list_config，若 `filter_type` 非空则调用 `validateTypeName` 校验
  - [x] 1.2.4 验证测试通过（运行：`go test ./internal/yaml/ -run TestValidateFilterType -v -count=1`，确认通过）
  - [x] 1.2.5 重构：确认错误路径信息包含 entity/resource 上下文

## 2. IR 构建器扩展

- [x] 2.1 buildList 读取 FilterType 并设置 Filter.Type  <!-- TDD 任务 -->
  - [x] 2.1.1 写失败测试：`internal/ir/builder_test.go` — 构建含 `list_config.filter_type: BookMetaFilter` 的 IR，断言 `List.Filter.Type == "BookMetaFilter"`；省略 filter_type 时断言 `List.Filter.Type == "string"`
  - [x] 2.1.2 验证测试失败（运行：`go test ./internal/ir/ -run TestBuildListFilterType -v -count=1`，确认失败原因是 Filter.Type 固定为 "string"）
  - [x] 2.1.3 写最小实现：`internal/ir/builder.go` — `buildList` 函数读取 `lc.FilterType`，非空时设置 `Filter.Type` 为该值，默认 `"string"`
  - [x] 2.1.4 验证测试通过（运行：`go test ./internal/ir/ -run TestBuildListFilterType -v -count=1`，确认通过）
  - [x] 2.1.5 重构：确认字段号仍为 3，不受 filter_type 影响

## 3. example 扩展

- [x] 3.1 移除 api.yaml 中 meta 资源的 reader.http override  <!-- 非 TDD 任务 -->
  - [x] 3.1.1 执行变更：`examples/book/api.yaml` — 删除 meta 资源 `reader.http` 块（第 43-46 行），List 回归默认 POST
  - [x] 3.1.2 验证无回归（运行：`go run ./cmd/apigen generate -f examples/book/api.yaml`，确认 proto 生成成功）
  - [x] 3.1.3 检查：确认生成的 proto 中 List 注解为 `post: "..." body: "*"`

- [x] 3.2 api.yaml 声明 filter_type 示例  <!-- 非 TDD 任务 -->
  - [x] 3.2.1 执行变更：`examples/book/api.yaml` — meta 资源 `list_config` 下新增 `filter_type: BookMetaFilter`
  - [x] 3.2.2 执行变更：`examples/book/proto/demo/business/book/book.proto` — 新增 `BookMetaFilter` message 定义
  - [x] 3.2.3 验证无回归（运行：`go run ./cmd/apigen build -f examples/book/api.yaml`，确认编译成功）
  - [x] 3.2.4 检查：确认生成的 proto 中 List request 的 filter 字段类型为 `BookMetaFilter`

- [x] 3.3 扩展 e2e 测试覆盖 filter_type  <!-- TDD 任务 -->
  - [x] 3.3.1 写失败测试：`examples/book/e2e_grpc_test.go` 和 `e2e_http_test.go` — List 请求传入结构化 filter（`BookMetaFilter{Author: "..."}`），断言返回结果正确过滤
  - [x] 3.3.2 验证测试失败（运行：`cd examples/book && go test -run TestE2EListFilterType -v -count=1`，确认失败原因是 filter 字段类型不匹配）
  - [x] 3.3.3 写最小实现：重新生成 proto + 编译（`go run ./cmd/apigen build -f examples/book/api.yaml`），确认 filter 字段类型正确
  - [x] 3.3.4 验证测试通过（运行：`cd examples/book && go test ./... -v -count=1`，确认通过）
  - [x] 3.3.5 重构：整理测试用例

## 4. Documentation Sync (Required)

- [x] 4.1 sync design.md: record technical decisions, deviations, and implementation details after each code change
- [x] 4.2 sync tasks.md: 逐一检查所有顶层任务及其子任务，将已完成但仍为 `[ ]` 的条目标记为 `[x]`；每次更新只修改 `[ ]` → `[x]`，禁止修改任何任务描述文字
- [x] 4.3 sync proposal.md: update scope/impact if changed
- [x] 4.4 sync specs/*.md: update requirements if changed
- [x] 4.5 Final review: ensure all OpenSpec docs reflect actual implementation
