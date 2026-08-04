# List Entity-Level + Offset Pagination — 任务清单

## 1. YAML Schema 变更

- [x] 1.1 Entity 新增 List 字段，ReaderDef 移除 List/ListConfig  <!-- TDD 任务 -->
  - [x] 1.1.1 写失败测试：`internal/yaml/parser_test.go` — 解析含 `entity.list.resource: meta` 的 api.yaml，断言 `Entity.List.Resource == "meta"` 且 `Entity.List.ListConfig.FilterType == "BookMetaFilter"`；同时验证 `ReaderDef` 不再有 `List`/`ListConfig` 字段
  - [x] 1.1.2 验证测试失败（运行：`go test ./internal/yaml/ -run TestParseEntityList -v -count=1`，确认失败原因是 Entity 无 List 字段）
  - [x] 1.1.3 写最小实现：`internal/yaml/parser.go` — 新增 `EntityListDef` 结构体（`Resource string` + `ListConfig *ListConfig`），`Entity` 新增 `List *EntityListDef`；`ReaderDef` 移除 `List bool` 和 `ListConfig *ListConfig`
  - [x] 1.1.4 验证测试通过（运行：`go test ./internal/yaml/ -run TestParseEntityList -v -count=1`，确认通过）
  - [x] 1.1.5 重构：确认 yaml tag 一致性（`list,omitempty`）

- [x] 1.2 ListConfig 移除 TotalSize 字段  <!-- TDD 任务 -->
  - [x] 1.2.1 写失败测试：`internal/yaml/parser_test.go` — 解析含 `list_config.total_size: true` 的 api.yaml，确认 `ListConfig` 无 `TotalSize` 字段（编译时错误即为测试通过条件）
  - [x] 1.2.2 写最小实现：`internal/yaml/parser.go` — `ListConfig` 移除 `TotalSize bool` 字段，仅保留 `FilterType string`
  - [x] 1.2.3 验证所有 yaml 测试通过（运行：`go test ./internal/yaml/ -v -count=1`）

- [x] 1.3 validate 扩展 entity.list 校验  <!-- TDD 任务 -->
  - [x] 1.3.1 写失败测试：`internal/yaml/validate_test.go` — `entity.list.resource` 指向不存在的 resource 时断言返回错误；`entity.list.resource` 为空时断言返回错误
  - [x] 1.3.2 验证测试失败（运行：`go test ./internal/yaml/ -run TestValidateEntityList -v -count=1`）
  - [x] 1.3.3 写最小实现：`internal/yaml/validate.go` — 新增 entity 级 list 校验：`list.resource` 非空且必须是 entity 下已声明的 resource 名称；`list_config.filter_type` 校验复用现有 `validateTypeName`
  - [x] 1.3.4 验证测试通过（运行：`go test ./internal/yaml/ -run TestValidateEntityList -v -count=1`）
  - [x] 1.3.5 重构：确认错误路径信息包含 entity 上下文

## 2. IR 构建器变更

- [x] 2.1 ListIR 结构体变更：PageSize/PageToken→Limit/Offset，移除 NextPageToken，TotalSize 改必选  <!-- TDD 任务 -->
  - [x] 2.1.1 写失败测试：`internal/ir/builder_test.go` — 构建 List IR，断言 `List.Limit.Name == "limit"` 且 `List.Limit.Number == 1`，`List.Offset.Name == "offset"` 且 `List.Offset.Number == 2`，`List.TotalSize.Number == 2`（Response 字段），无 `NextPageToken` 字段
  - [x] 2.1.2 验证测试失败（运行：`go test ./internal/ir/ -run TestBuildListFields -v -count=1`）
  - [x] 2.1.3 写最小实现：`internal/ir/builder.go` — `ListIR` 结构体变更：`PageSize`→`Limit`，`PageToken`→`Offset`，移除 `NextPageToken`，`TotalSize` 从 `*FieldIR` 改为 `FieldIR`
  - [x] 2.1.4 验证测试通过（运行：`go test ./internal/ir/ -run TestBuildListFields -v -count=1`）

- [x] 2.2 List 从 ResourceIR 移到 EntityIR  <!-- TDD 任务 -->
  - [x] 2.2.1 写失败测试：`internal/ir/builder_test.go` — 构建含 `entity.list.resource: meta` 的 IR，断言 `EntityIR.List != nil` 且 `EntityIR.List.RPCName == "ListBookMetas"`；同时断言 `ResourceIR` 不再有 `List` 字段
  - [x] 2.2.2 验证测试失败（运行：`go test ./internal/ir/ -run TestBuildEntityList -v -count=1`）
  - [x] 2.2.3 写最小实现：`internal/ir/builder.go` — `EntityIR` 新增 `List *ListIR`；`ResourceIR` 移除 `List *ListIR`；`buildEntity()` 中根据 `e.List` 构建 List IR（查找 `e.List.Resource` 对应的 resource 类型）；`buildResource()` 移除 List 构建逻辑
  - [x] 2.2.4 验证测试通过（运行：`go test ./internal/ir/ -run TestBuildEntityList -v -count=1`）
  - [x] 2.2.5 重构：确认 `buildList()` 函数签名调整（参数从 resource 名/类型改为 entity 名 + resource 名/类型）

- [x] 2.3 List HTTP 注解移到 entity 级  <!-- TDD 任务 -->
  - [x] 2.3.1 写失败测试：`internal/ir/builder_test.go` — 构建含 HTTP enabled 的 List IR，断言 `EntityIR.List.HTTPAnnotation.Verb == "POST"`，`HTTPAnnotation.Entity == "book"`，`HTTPAnnotation.Resource == ""`（无 resource 段），`HTTPAnnotation.Suffix == "list"`；调用 `ResolvePath("/library", "LibraryService")` 断言路径为 `/library/LibraryService/book/list`
  - [x] 2.3.2 验证测试失败（运行：`go test ./internal/ir/ -run TestBuildListHTTP -v -count=1`）
  - [x] 2.3.3 写最小实现：`internal/ir/builder.go` — 新增 `buildListAnnotation()` 函数（entity 级，无 KeyLeaves，无 Resource 段，Suffix="list"）；`buildEntity()` 中调用；`fillResourceAnnotations` 移除 List 相关代码
  - [x] 2.3.4 验证测试通过（运行：`go test ./internal/ir/ -run TestBuildListHTTP -v -count=1`）

- [x] 2.4 Service 收窄机制适配  <!-- TDD 任务 -->
  - [x] 2.4.1 写失败测试：`internal/ir/builder_test.go` — 构建含 service entity `list: false` 的 IR，断言 `ServiceEntityIR.List` 为 `*bool(false)`；`narrowEntity` 后 `EntityIR.List == nil`
  - [x] 2.4.2 验证测试失败（运行：`go test ./internal/ir/ -run TestNarrowList -v -count=1`）
  - [x] 2.4.3 写最小实现：`internal/ir/builder.go` — `ServiceEntityIR` 新增 `List *bool`；`buildService()` 解析 service entity 的 `list` 字段；`ReaderNarrowIR` 移除 `List *bool`
  - [x] 2.4.4 验证测试通过（运行：`go test ./internal/ir/ -run TestNarrowList -v -count=1`）

## 3. 渲染层变更

- [x] 3.1 List 渲染从 resource 循环移到 entity 级  <!-- TDD 任务 -->
  - [x] 3.1.1 写失败测试：`internal/render/template_test.go` — 渲染含 entity.list 的 proto，断言 List RPC 出现在 service 定义中，List message（Request/Response）出现在 message 定义中；Request 含 `limit=1, offset=2, filter=3, order_by=4`，Response 含 `<resource>s=1, total_size=2`，无 `next_page_token`
  - [x] 3.1.2 验证测试失败（运行：`go test ./internal/render/ -run TestRenderEntityList -v -count=1`）
  - [x] 3.1.3 写最小实现：`internal/render/template.go` — `renderServiceRPCs` 中 entity 级 RPC 部分新增 List 渲染；`renderMessages` 中 entity 级 message 部分新增 List message 渲染；移除 resource 循环中的 List 渲染代码
  - [x] 3.1.4 验证测试通过（运行：`go test ./internal/render/ -run TestRenderEntityList -v -count=1`）
  - [x] 3.1.5 重构：确认 List message 模板使用 `Limit`/`Offset`/`TotalSize`（非 `PageSize`/`PageToken`/`NextPageToken`）

- [x] 3.2 narrowEntity 适配 entity 级 List 收窄  <!-- TDD 任务 -->
  - [x] 3.2.1 写失败测试：`internal/render/template_test.go` — `narrowEntity` 传入 `ServiceEntityIR{List: boolPtr(false)}`，断言返回的 `EntityIR.List == nil`；传入 `List: boolPtr(true)` 或 `nil`，断言 `EntityIR.List != nil`
  - [x] 3.2.2 验证测试失败（运行：`go test ./internal/render/ -run TestNarrowEntityList -v -count=1`）
  - [x] 3.2.3 写最小实现：`internal/render/template.go` — `narrowEntity` 新增 List 收窄逻辑：`se.List != nil && !*se.List` 时 `out.List = nil`；移除 resource 级 `ReaderNarrowIR.List` 收窄代码
  - [x] 3.2.4 验证测试通过（运行：`go test ./internal/render/ -run TestNarrowEntityList -v -count=1`）

## 4. Example 与 Testcase 更新

- [x] 4.1 更新 examples/book/api.yaml  <!-- 非 TDD 任务 -->
  - [x] 4.1.1 执行变更：`examples/book/api.yaml` — book entity 新增 `list: { resource: meta, list_config: { filter_type: BookMetaFilter } }`；meta resource 的 `reader` 移除 `list: true` 和 `list_config`
  - [x] 4.1.2 执行变更：`examples/book/api.yaml` — AdminService 的 book entity 收窄从 `resources: [{ name: meta, reader: { list: true } }]` 改为 `list: true`
  - [x] 4.1.3 验证无回归（运行：`go run ./cmd/apigen generate -f examples/book/api.yaml`，确认 proto 生成成功）

- [x] 4.2 更新 testcase/fixtures/book/api.yaml  <!-- 非 TDD 任务 -->
  - [x] 4.2.1 执行变更：`testcase/fixtures/book/api.yaml` — 同 4.1 的变更应用到 testcase fixture
  - [x] 4.2.2 更新所有 testcase golden 文件（运行 testcase 更新命令，确认生成的 proto/Go 产物与 golden 一致）

- [x] 4.3 更新 testcase 中 invalid YAML fixtures  <!-- 非 TDD 任务 -->
  - [x] 4.3.1 检查 `testcase/fixtures/invalid/filter_type_invalid.yaml` — 如果使用了 `resource.reader.list_config`，需改为 `entity.list.list_config`
  - [x] 4.3.2 新增 invalid fixture：`entity.list.resource` 指向不存在的 resource 时应报错

- [x] 4.4 重新生成 examples/book 产物  <!-- 非 TDD 任务 -->
  - [x] 4.4.1 运行：`go run ./cmd/apigen build -f examples/book/api.yaml`
  - [x] 4.4.2 检查生成的 proto 中 List Request 含 `limit=1, offset=2`，Response 含 `total_size=2`，无 `next_page_token`
  - [x] 4.4.3 检查 HTTP 注解路径为 `/{prefix}/{svc}/{entity}/list`

- [x] 4.5 更新 e2e 测试  <!-- TDD 任务 -->
  - [x] 4.5.1 写失败测试：`examples/book/e2e_grpc_test.go` 和 `e2e_http_test.go` — List 请求使用 `Limit`/`Offset` 字段（非 `PageSize`/`PageToken`），断言 Response 无 `NextPageToken`，有 `TotalSize`
  - [x] 4.5.2 验证测试失败（运行：`cd examples/book && go test -run TestE2EList -v -count=1`）
  - [x] 4.5.3 写最小实现：重新生成 proto + 编译，确认字段名正确
  - [x] 4.5.4 验证测试通过（运行：`cd examples/book && go test ./... -v -count=1`）

## 5. Documentation Sync (Required)

- [x] 5.1 sync design.md: record technical decisions, deviations, and implementation details after each code change
- [x] 5.2 sync tasks.md: 逐一检查所有顶层任务及其子任务，将已完成但仍为 `[ ]` 的条目标记为 `[x]`；每次更新只修改 `[ ]` → `[x]`，禁止修改任何任务描述文字
- [x] 5.3 sync proposal.md: update scope/impact if changed
- [x] 5.4 sync specs/*.md: update requirements if changed
- [x] 5.5 Final review: ensure all OpenSpec docs reflect actual implementation
