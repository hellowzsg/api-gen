## Why

当前 List 方法的 `filter` 字段固定为 `string` 类型（design-v2.md 决策 #34 / D12），filter 语义完全由服务端解释。实际业务中，很多场景的过滤条件是结构化的（如 `BookMetaFilter { author, year_from, year_to, tags }`），用 string 编码 JSON 或 AIP-160 表达式增加了客户端序列化/反序列化负担，且丧失了 proto 的类型安全优势。

同时，当前 `examples/book/api.yaml` 中 meta 资源的 `reader.http` override 将 List 改为 GET + query param 传参。但 GET 方式下结构化 filter 只能序列化为 string 塞进 URL query，有 URL 长度限制和编码问题。如果 List 使用默认的 POST + `body:"*"`，结构化 filter message 可以直接作为 request 字段通过 body 传递，无需序列化。

本提案解除 D12 冻结，允许 `list_config.filter_type` 指定自定义 message 类型，并将 example 中 List 的 `reader.http` override 移除（回归默认 POST）。

## What Changes

### 1. `list_config.filter_type` 新增字段

`reader.list_config` 新增可选字段 `filter_type`，类型为 string（proto FQMN 或短名）。

- 省略或为空：`filter` 字段类型为 `string`（完全向后兼容）
- 指定值（如 `BookMetaFilter`）：`filter` 字段类型为该 message 类型

```yaml
resources:
  - name: meta
    reader:
      list: true
      list_config:
        total_size: true
        filter_type: BookMetaFilter   # filter 字段类型变为 BookMetaFilter
```

### 2. 移除 example 中 List 的 `reader.http` override

`examples/book/api.yaml` 中 meta 资源的 `reader.http` override（将 List 改为 GET）不再需要，移除后 List 回归默认 POST + `body:"*"`，filter 走 body 传递。

### 3. filter_type 可达性校验

`filter_type` 指向的 message 类型需在 import 闭包中可达。校验策略与 `custom_method.request`/`response` 一致：YAML validate 阶段做语法校验（非空、合法标识符），类型可达性由 protocompile link 阶段兜底。

### 不在本次范围（Non-Goals）

- `order_by` 的类型定制（保持 `string`，排序语义标准化程度高）
- `page_size`/`page_token` 的类型定制（保持标量）
- filter_type 指向类型的内部结构校验（与 key type 一样，完全黑盒）
- service 级别的 filter_type 覆盖（entity 级声明即可，所有继承该 entity 的 service 共享同一 filter 类型）

## Impact

### 新增代码

- `internal/yaml/parser.go`：`ListConfig` 新增 `FilterType string` 字段
- `internal/ir/builder.go`：`buildList` 读取 `lc.FilterType`，非空时设置 `Filter.Type` 为该值（默认 `"string"`）
- `internal/yaml/validate.go`：`validateTypeReferences` 扩展，对 `filter_type` 做语法校验（复用 `validateTypeName`）

### 变更代码

- `examples/book/api.yaml`：移除 meta 资源的 `reader.http` override（第 43-46 行）

### 用户工作区产物

- 生成的 proto 中 List request 的 `filter` 字段类型从 `string` 变为用户指定的 message 类型（仅当声明了 `filter_type` 时）

### 向后兼容性

- `filter_type` 省略时行为完全不变（`filter` 仍为 `string`）
- 移除 example 的 `reader.http` override 不影响 API 契约（仅 HTTP verb 从 GET 回归 POST，path 不变）

### example 扩展

- `examples/book/api.yaml`：meta 资源声明 `filter_type`（如使用示例 proto 中已有的类型或新增 `BookMetaFilter` message）
- `examples/book/proto/`：如有需要，新增 `BookMetaFilter` message 定义
- `examples/book/` e2e 测试：覆盖 filter_type 为自定义 message 时的 List 行为
