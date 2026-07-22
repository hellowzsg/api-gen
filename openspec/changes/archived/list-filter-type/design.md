## Context

当前 List 方法的 `filter` 字段固定为 `string` 类型（design-v2.md 决策 #34 / D12）。决策理由是"filter 语义因业务而异，工具不预设结构"。但实际业务中结构化 filter 需求强烈，且 List 默认使用 POST + `body:"*"`，结构化 message 可以直接通过 body 传递，无需序列化为 string。

当前代码现状：
- `internal/yaml/parser.go`：`ListConfig` 仅有 `TotalSize bool` 字段
- `internal/ir/builder.go`：`buildList` 固定 `Filter: FieldIR{Name: "filter", Type: "string", Number: 3}`
- `internal/render/template.go`：渲染 `fmt.Sprintf("  %s filter = %d;\n", r.List.Filter.Type, r.List.Filter.Number)`——已经是用 `Filter.Type` 间接引用类型，无需改动
- `examples/book/api.yaml`：meta 资源声明了 `reader.http: { verb: get, path: ... }` 将 List 改为 GET

## Goals / Non-Goals

**Goals:**
1. `list_config.filter_type` 支持指定自定义 message 类型作为 filter 字段类型
2. 移除 example 中不必要的 `reader.http` override，List 回归默认 POST
3. filter_type 省略时完全向后兼容（`filter` 为 `string`）

**Non-Goals:**
- `order_by` 类型定制（保持 `string`）
- filter_type 指向类型的内部结构校验（黑盒，与 key type 一致）
- service 级 filter_type 覆盖

## Decisions

### 决策 1：filter_type 声明位置

- **位置**：`reader.list_config.filter_type`，与 `total_size` 同级
- **理由**：filter 是 List 专属字段（BatchGet/Get 无 filter），放在 `list_config` 下语义最精确；不放在 `reader` 顶层避免与 `reader.http`（HTTP 覆盖）混淆

### 决策 2：filter_type 值格式

- **格式**：短名（如 `BookMetaFilter`）或全限定名（如 `demo.business.book.BookMetaFilter`），与 `custom_method.request`/`response` 一致
- **渲染**：直接作为 proto 字段类型输出，由 protocompile link 阶段解析可达性
- **不自动补包名前缀**：与现有 `type_`/`request`/`response` 字段处理方式一致

### 决策 3：可达性校验策略

- **YAML validate 阶段**：复用 `validateTypeName` 做语法校验（非空、合法标识符开头）
- **类型可达性**：由 protocompile link 阶段兜底（与 custom_method request/response 一致，design-v2.md §15.4 残余风险）
- **不做 filter_type 指向类型的内部结构校验**：与 key type 黑盒策略一致

### 决策 4：移除 example 的 reader.http override

- **当前**：`reader.http: { verb: get, path: /library/LibraryService/book/meta/list }` 将 List 改为 GET
- **变更后**：移除该 override，List 回归默认 POST + `body:"*"`，path 仍为 `/library/{svc}/book/meta/list`
- **理由**：POST + body 传递结构化 filter 更自然；GET + query param 只适合 string filter 且有 URL 长度限制

### 决策 5：字段号不变

- `filter` 字段号固定为 3（design-v2.md 冻结约定）
- 类型从 `string` 变为自定义 message **不改变字段号**，仅改变类型声明
- 对 wire format 的影响：proto3 message 字段与 string 字段在 wire format 上不同（message 字段有 length-delimited 前缀），因此这是**不兼容的 wire 变更**——但这是用户显式选择的结构化 filter，由用户承担兼容性责任

## Risks / Trade-offs

1. **wire 不兼容**：filter 从 `string` 改为 message 类型是 wire-level 不兼容变更。用户需在切换 filter_type 时确保客户端同步升级。这是用户显式选择的，工具不做静默转换。
2. **filter_type 可达性延迟到编译期**：与 custom_method 一致的残余风险，YAML validate 不做闭包校验，protocompile link 阶段报错。错误信息可能不够友好（指向 proto 编译错误而非 api.yaml 位置），但可通过错误信息中的类型名溯源。
3. **不校验 filter_type 指向类型的内部结构**：与 key type 黑盒策略一致，业务自行保证 filter message 的字段设计合理。

## Implementation Notes

### 渲染模板修正

实现过程中发现 `internal/render/template.go` 第 175 行硬编码了 `string filter`：
```go
// 修改前（硬编码）：
sb.WriteString(fmt.Sprintf("  string filter = %d;\n", r.List.Filter.Number))
// 修改后（使用 Filter.Type）：
sb.WriteString(fmt.Sprintf("  %s filter = %d;\n", r.List.Filter.Type, r.List.Filter.Number))
```
这是提案未预见的改动——原以为渲染层已经通过 `Filter.Type` 间接引用，但实际模板硬编码了 `"string"`。修正后渲染层正确使用 IR 中的 `Filter.Type`，向后兼容（`Filter.Type` 默认为 `"string"`）。

### example 变更总结

- `examples/book/api.yaml`：移除 `reader.http` override（List 从 GET 回归 POST），新增 `list_config.filter_type: BookMetaFilter`
- `examples/book/proto/demo/business/book/book.proto`：新增 `BookMetaFilter` message（author/title/year_from/year_to/tags 五个字段）
- `examples/book/e2e_grpc_test.go`：ListBookMetas 测试的 `Filter` 字段从 `string` 改为 `*bookpb.BookMetaFilter`
- `examples/book/e2e_http_test.go`：List HTTP 测试从 GET + query param 改为 POST + body JSON；AdminService List 同步改为 POST
- 生成的 proto/go/js/openapi 产物已通过 `apigen build` 重新生成
