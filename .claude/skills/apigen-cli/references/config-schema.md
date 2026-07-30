# api.yaml 配置 Schema

## 根结构

```yaml
syntax: v1              # 必填，配置格式标识
name: demo.business.book  # 必填，点分 proto package 名
import_protos: []       # 依赖来源
settings: {}             # 输出与生成行为
entities: []             # 必填，领域模型（至少一个实体）
services: []             # 服务暴露
```

解析为严格模式 — 未知字段直接报错。

## import_protos

每项选择以下一种来源：

| 字段 | 说明 |
| --- | --- |
| `path` | 本地 proto glob，相对 `api.yaml` 目录解析 |
| `git` | Git 仓库 URL；配合 `ref`（分支/标签/commit）和 `subdir` 使用 |
| `bsr` | BSR 模块名，如 `buf.build/googleapis/googleapis` |
| `alias` | 兼容字段；不用于 type_ 别名解析 |
| `version` | BSR 条目的兼容字段；不用于固定版本 |

示例：

```yaml
import_protos:
  - path: "proto/**/*.proto"
  - git: https://github.com/googleapis/googleapis
    ref: master
    subdir: google
  - bsr: buf.build/googleapis/googleapis
```

启用 HTTP 时，依赖中必须可解析 `google/api/annotations.proto` 及其传递引用。

## settings

| 字段 | 类型 | 作用 |
| --- | --- | --- |
| `go_repo` | string | 写入生成 proto 的 `go_package` 的 Go module path |
| `js_repo` | string | 兼容字段；不影响 TS 输出 |
| `out.proto` | string | 生成 service `.proto` 文件的目录 |
| `out.go` | string | Go 代码输出目录 |
| `out.js` | string | TypeScript 输出目录 |
| `out.openapi` | string | OpenAPI v2 输出目录 |
| `http.enable` | bool | 启用 `google.api.http` 注解和 grpc-gateway |
| `http.prefix` | string | 自动生成 HTTP 路由的全局前缀，如 `/api` |
| `http.body_style` | string | 默认 HTTP body 策略：`wrapper`（默认，= `body:"*"`）或 `resource` |
| `http.generate_openapi` | bool | 生成 OpenAPI v2 文档（需 HTTP 启用 + `out.openapi` 已设置） |
| `plugins.js` | []string | JS 插件列表；当前仅支持 `es` |

## entities

每个实体必须包含主键和至少一个资源。

### 实体字段

| 字段 | 类型 | 作用 |
| --- | --- | --- |
| `name` | string | 实体名，`snake_case`；生成类型的命名词干 |
| `key.type_` | string | 主键 message 类型；可使用全限定名 |
| `create` | object | 生成 `Create`；支持 `key` 子字段 |
| `create.key` | string | 主键提供方：`server`（默认）或 `client` |
| `delete` | object | 设为 `{}` 生成硬删除 `Delete` |
| `delete_soft` | object | 设为 `{}` 生成软删除 `DeleteSoft`；可与 `delete` 并存 |
| `resources` | []object | 至少声明一个资源 |

### 资源字段

| 字段 | 类型 | 作用 |
| --- | --- | --- |
| `name` | string | 资源名，`snake_case` |
| `type_` | string | 资源 message 类型；可使用全限定名 |
| `version.kind` | string | 并发控制：`STRONG` / `WEAK` / `NONE` |
| `version.type` | string | 版本值类型：`U64` / `U32` / `STRING`（STRONG/WEAK 时需要） |
| `reader` | object | 读取能力。`reader: {}` 生成 `Get` |
| `reader.batch` | bool | 生成 `BatchGet` |
| `reader.list` | bool | 生成 `List`（含分页、filter、order_by） |
| `reader.list_config.total_size` | bool | List 响应是否包含 `total_size`（默认 true） |
| `reader.list_config.filter_type` | string | List 请求的自定义 filter message 类型 |
| `reader.http` | object | 覆盖 List 的 HTTP `verb`/`path`/`body`/`body_style` |
| `writer.update` | object | 生成 `Update` |
| `writer.update.mask` | bool | Update 请求中是否包含 `google.protobuf.FieldMask` |
| `writer.update.http` | object | 覆盖 Update 的 HTTP `verb`/`path`/`body`/`body_style` |
| `options` | []object | 预留字段；不写入生成的 proto option |

### 版本策略

| 策略 | Get 响应 | Update 请求 | Update 响应 | 使用场景 |
| --- | --- | --- | --- | --- |
| `STRONG` | 标量 `version` | 标量 `version`（强制 CAS） | 更新后的标量版本 | 强制 CAS |
| `WEAK` | wrapper `version` | 可选 wrapper `version` | 更新后的 wrapper 版本 | 可选 CAS |
| `NONE` | 无版本 | 无版本 | `google.protobuf.Empty` | 无乐观锁 |

`STRONG` 的标量类型和 `WEAK` 的 wrapper 类型由 `version.type` 决定：`U64`、`U32` 或 `STRING`。

## services

| 字段 | 类型 | 作用 |
| --- | --- | --- |
| `services[].name` | string | Service 名，`PascalCase` |
| `services[].entities[].name` | string | 引用 `entities` 中定义的实体 |
| `services[].entities[].resources` | []object | 可选的收窄规则；省略时暴露实体全部能力 |
| `services[].custom_methods` | []object | service 级自定义 RPC 列表 |
| `custom_methods[].name` | string | RPC 名，`PascalCase` |
| `custom_methods[].request` | string | Request message 类型 |
| `custom_methods[].response` | string | Response message 类型 |
| `custom_methods[].http` | object | HTTP 启用时可设置 `verb`、`path`（AIP-136 冒号语法）、`body` |

示例：

```yaml
services:
  - name: LibraryService
    entities:
      - name: book                 # 暴露全部能力

  - name: AdminService
    entities:
      - name: book
        resources:
          - name: meta
            reader: { list: true } # 仅暴露 ListBookMetas
    custom_methods:
      - name: ArchiveBook
        request: ArchiveBookRequest
        response: ArchiveBookResponse
        http:
          verb: post
          path: /library/books/{book_id}:archive
          body: "*"
```

## api.lock

位于 `api.yaml` 同目录。记录 Git 依赖的 resolved commit，确保可复现构建。

```json
[
  {
    "url": "https://github.com/googleapis/googleapis",
    "ref": "master",
    "resolved_commit": "abc123def456..."
  }
]
```

BSR 依赖不记录在 `api.lock` 中（由 buf 内部管理）。
