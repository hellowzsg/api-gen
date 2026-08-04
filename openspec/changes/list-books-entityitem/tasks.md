# List Books + EntityItem — 任务清单

## 1. YAML Schema 变更

- [x] 1.1 EntityListDef.Resource 改为 []string
  - [x] 1.1.1 修改 `internal/yaml/parser.go`：`EntityListDef.Resource string` → `Resources []string`（yaml tag `resource`）
  - [x] 1.1.2 更新 parser 测试：解析 `resource: [meta]` 断言 `Entity.List.Resources` 长度正确
- [x] 1.2 validate 扩展多资源校验
  - [x] 1.2.1 修改 `internal/yaml/validate.go`：`list.resource` 列表至少一个元素；每个元素必须是实体已声明资源；元素不可重复；空元素报错
  - [x] 1.2.2 更新 validate 测试与新增 invalid fixtures：空列表、不存在资源、重复元素、空元素

## 2. IR 构建器变更

- [x] 2.1 ListIR 结构调整
  - [x] 2.1.1 修改 `internal/ir/builder.go`：`ListIR` 移除 `ResourcesField`，新增 `ItemName string` + `ItemFields []FieldIR`
  - [x] 2.1.2 修改 `buildList(entityPascal string, resources []apigenyaml.Resource, cfg *apigenyaml.Config, lc *ListConfig)`：遍历 resources 构建 `ItemFields`（字段号 1..N），`ItemName` = `<Entity>Item`
  - [x] 2.1.3 更新 IR 测试：断言 `List.RPCName == "ListBooks"`、`ItemName == "BookItem"`、`ItemFields` 含 meta/content 两字段

## 3. 渲染层变更

- [x] 3.1 List Request/Response/Item 渲染
  - [x] 3.1.1 修改 `internal/render/template.go`：List Response 渲染 `repeated <ItemName> items = 1` + `total_size = 2`；新增 `<ItemName>` message 渲染（遍历 `ItemFields`）
  - [x] 3.1.2 更新 render 测试：渲染含多资源 List 的 proto，断言含 `ListBooksRequest`、`ListBooksResponse`、`BookItem`，且 `BookItem` 含 `meta=1, content=2`

## 4. Example / Testcase / e2e 更新

- [x] 4.1 更新 examples/book/api.yaml：`list.resource: [meta, content]`
- [x] 4.2 更新 testcase/fixtures/book/api.yaml：同 4.1；AdminService 收窄 resources 至 meta（无 batch），使其体现收窄语义
- [x] 4.3 更新 testcase invalid fixtures：新增 `list_resource_empty.yaml`、`list_resource_duplicate.yaml`，调整 `list_resource_not_found.yaml` 为列表形式
- [x] 4.4 重新生成 examples/book 与 fixtures/book 产物：`apigen build`
- [x] 4.5 更新 e2e 测试：`ListBooks` RPC、`BookItem` Response、`items` 字段
- [x] 4.6 更新 cli 测试中 List 相关断言（`ListBookMetas` → `ListBooks`）

## 5. Documentation Sync

- [x] 5.1 更新 design.md：记录实现偏差
- [x] 5.2 更新 tasks.md：标记已完成任务
- [x] 5.3 更新 specs/apigen.md：同步 Requirements
- [x] 5.4 更新 README.md / README_EN.md / examples/README.md / design-v2.md
- [x] 5.5 更新 .claude/skills/apigen-cli/ 及其 references

## 6. 验证

- [x] 6.1 `go test ./internal/... -count=1` 全绿
- [x] 6.2 `cd examples/book && go test ./... -count=1` 全绿
- [x] 6.3 `cd testcase && go test ./... -count=1` 全绿
