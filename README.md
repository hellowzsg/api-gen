# apigen

> 基于实体模型生成 AIP 风格 gRPC 服务定义，并按需生成 Go、HTTP 网关、OpenAPI 和 TypeScript 代码。

[English](README_EN.md) | 简体中文

## 项目简介

`apigen` 是一个声明式 API 生成工具。开发者在 `api.yaml` 中描述业务实体的主键、资源数据面、读写能力、并发控制与服务暴露范围；工具据此生成服务层 `.proto`、标准 Request/Response、分页与更新掩码等样板代码，并可进一步编译为多语言客户端代码。

## 功能概览

- **声明式定义**：通过 `api.yaml` 描述实体、资源、读写策略与服务，无需手写样板 proto。
- **标准 gRPC API 生成**：自动生成符合 AIP 规范的 Create、Delete、Get、BatchGet、List、Update 等方法。
- **乐观锁支持**：内置版本控制策略，支持 CAS 并发更新。
- **HTTP 转码**：一键生成 `google.api.http` 注解和 grpc-gateway 反向代理代码。
- **多语言客户端**：支持生成 Go、TypeScript stub 及 OpenAPI v2 接口文档。
- **依赖管理**：可引用本地、Git 仓库或 BSR 中的 proto 类型，并支持锁文件固定版本。

## 环境要求

| 项目 | 要求 | 适用场景 |
|---|---|---|
| Go | **1.24+** | 安装、运行 `apigen` |
| `protoc-gen-go` | 必需 | 生成 Go message 代码 |
| `protoc-gen-go-grpc` | 必需 | 生成 Go gRPC 代码 |
| `protoc-gen-grpc-gateway` | 启用 HTTP 时必需 | 生成 `*.pb.gw.go` |
| `protoc-gen-openapiv2` | 生成 OpenAPI 时必需 | 生成 Swagger 文档 |
| `protoc-gen-es` | 生成 TypeScript 时必需 | 生成 `*.pb.ts` |
| Git CLI | 使用 `import_protos.git` 时必需 | 拉取 Git proto 依赖 |
| Buf CLI | 使用 `import_protos.bsr` 时必需 | 导出 BSR 模块 |

> `apigen build` 通过 Protobuf 插件协议调用生成器，**无需单独安装 `protoc`**。请确保所需插件均在 `PATH` 中。

## 安装与快速开始

### 1. 安装 CLI

```bash
git clone <repository-url> aip-gen
cd aip-gen
go install ./cmd/apigen

# 验证安装
apigen --help
```

### 2. 安装生成 Go 代码所需插件

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

如需 HTTP、OpenAPI 或 TypeScript，再按需安装对应插件：

```bash
# HTTP 转码与 OpenAPI
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

# TypeScript
npm install --global @bufbuild/protoc-gen-es
```

### 3. 运行内置示例

仓库中的 `examples/book/` 覆盖实体、资源、版本控制、HTTP、OpenAPI、TypeScript 与 service 收窄等能力：

```bash
# 仅查看将生成的实体和方法，不写文件
go run ./cmd/apigen entity list -f examples/book/api.yaml

# 生成 service proto
go run ./cmd/apigen generate -f examples/book/api.yaml

# 生成 proto 并编译 Go / HTTP / OpenAPI / TypeScript 产物
go run ./cmd/apigen build -f examples/book/api.yaml
```

生成后的主要文件位于 `examples/book/generated/`。更多示例说明见 [examples/README.md](examples/README.md)。

## 使用示例

### 定义领域类型

`apigen` 生成服务层协议；实体主键和资源 message 由你维护。以下为 `proto/demo/business/book/book.proto` 的简化示例：

```proto
syntax = "proto3";

package demo.business.book;

option go_package = "github.com/acme/demo-book/generated/go/demo/business/book;book";

message BookId {
  string id = 1;
}

message BookMeta {
  string title = 1;
  string author = 2;
}
```

### 编写 `api.yaml`

```yaml
syntax: v1
name: demo.business.book

import_protos:
  - path: "proto/**/*.proto"

settings:
  go_repo: github.com/acme/demo-book
  out:
    proto: generated/proto
    go: generated/go

entities:
  - name: book
    key: { type_: BookId }
    create: {}
    delete: {}
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: STRONG, type: U64 }
        reader: { batch: true, list: true }
        writer:
          update: { mask: true }

services:
  - name: LibraryService
    entities:
      - name: book
```

### 生成与编译

```bash
# 仅生成 proto
apigen generate -f api.yaml

# 生成 proto，并编译 Go 代码
apigen build -f api.yaml

# 查看实体、资源和方法清单
apigen entity list -f api.yaml
```

上述配置会生成 `LibraryService`，包含 `CreateBook`、`DeleteBook`、`GetBookMeta`、`BatchGetBookMetas`、`ListBookMetas` 和 `UpdateBookMeta` 等方法。

## 命令参考

所有命令均可使用 `--file` 或 `-f` 指定配置文件，默认读取当前目录的 `api.yaml`。

| 命令 | 说明 |
|---|---|
| `apigen generate -f api.yaml` | 校验配置与依赖，生成各 service 的 `.proto` 文件。 |
| `apigen build -f api.yaml` | 先执行 `generate`，再调用已安装的代码生成插件。 |
| `apigen entity list -f api.yaml` | 输出实体、资源与将生成的方法，不写入文件。 |
| `apigen dep update -f api.yaml` | 刷新远程依赖，并更新 Git 依赖的 `api.lock` 记录。 |
| `apigen dep prune -f api.yaml` | 预留的依赖清理命令；当前版本不会移除现有 lock 条目。 |

## `api.yaml` 配置说明

### 配置结构

```yaml
syntax: v1
name: example.catalog

import_protos: []
settings: {}
entities: []
services: []
```

配置会以严格字段模式解析；拼写错误或未知字段会直接报错。`entities` 是必填项，每个实体必须包含主键与至少一个资源。

### 根字段与依赖来源

| 字段 | 类型 | 作用 |
|---|---|---|
| `syntax` | `string` | 配置格式标识。示例使用 `v1`。 |
| `name` | `string` | 业务 proto package，格式为点分标识符，如 `demo.business.book`。 |
| `import_protos` | `[]object` | 声明 `key.type_` 与资源 `type_` 对应的 proto 来源。 |
| `settings` | `object` | 控制输出路径、HTTP 与语言插件。 |
| `entities` | `[]object` | 定义业务实体、资源和读写能力。 |
| `services` | `[]object` | 定义对外暴露的 gRPC service。 |

`import_protos` 的每项可选择下列一种来源：

| 字段 | 说明 |
|---|---|
| `path` | 本地 proto glob；相对 `api.yaml` 所在目录解析。 |
| `git` | Git 仓库 URL。可配合 `ref`（分支、标签或 commit）与 `subdir`（仓库内 proto 子目录）。 |
| `bsr` | BSR 模块名，如 `buf.build/googleapis/googleapis`。 |
| `alias` | 已接受的兼容字段；当前版本不用于 `type_` 的别名解析。 |
| `version` | BSR 条目可接受的兼容字段；当前导出流程不使用该值固定版本。 |

示例：

```yaml
import_protos:
  - path: "proto/**/*.proto"
  - git: https://github.com/googleapis/googleapis
    ref: master
    subdir: google
  - bsr: buf.build/googleapis/googleapis
```

> 启用 HTTP 时，依赖中必须可解析 `google/api/annotations.proto` 与其关联定义；可通过本地 vendored proto、Git 或 BSR 提供。

### `settings`：输出与生成行为

| 字段 | 类型 | 作用 |
|---|---|---|
| `go_repo` | `string` | 写入生成 proto 的 `go_package` 的 Go module path。 |
| `js_repo` | `string` | 已接受的兼容字段；当前不会影响 TypeScript 生成结果。 |
| `out.proto` | `string` | 生成 service `.proto` 的目录。 |
| `out.go` | `string` | Go 代码输出目录。 |
| `out.js` | `string` | TypeScript 输出目录。 |
| `out.openapi` | `string` | OpenAPI v2 输出目录。 |
| `http.enable` | `bool` | 开启 `google.api.http` 注解与 grpc-gateway 代码生成。 |
| `http.prefix` | `string` | 自动生成 HTTP 路由的全局前缀，如 `/api`。 |
| `http.body_style` | `string` | HTTP body 默认策略：`wrapper`（默认，等价于 `body: "*"`）或 `resource`。 |
| `http.generate_openapi` | `bool` | 是否生成 OpenAPI v2 文档。仅在 HTTP 开启且配置 `out.openapi` 时生效。 |
| `plugins.js` | `[]string` | JavaScript 插件列表；当前仅支持 `es`。 |

```yaml
settings:
  go_repo: github.com/acme/demo-book
  out:
    proto: generated/proto
    go: generated/go
    js: generated/js
    openapi: generated/openapi
  http:
    enable: true
    prefix: /library
    body_style: wrapper
    generate_openapi: true
  plugins:
    js: [es]
```

### `entities`：领域模型与操作

实体由主键和一个或多个资源组成。资源可视为同一业务对象的独立数据面，例如图书的 `meta`（元数据）和 `content`（正文）。`key.type_` 与资源 `type_` 必须引用已导入的、由开发者维护的 proto message。

#### 实体字段

| 字段 | 类型 | 作用 |
|---|---|---|
| `name` | `string` | 实体名，使用 `snake_case`，作为生成类型和方法的命名词干。 |
| `key.type_` | `string` | 主键 message 类型；可使用全限定名。 |
| `create` | `object` | 设为 `{}` 时生成 `Create`，响应仅携带 key。 |
| `delete` | `object` | 设为 `{}` 时生成硬删除 `Delete`。 |
| `delete_soft` | `object` | 设为 `{}` 时生成软删除 `DeleteSoft`；可与 `delete` 并存。 |
| `resources` | `[]object` | 至少声明一个资源。 |

#### 资源字段

| 字段 | 类型 | 作用 |
|---|---|---|
| `name` | `string` | 资源名，使用 `snake_case`。 |
| `type_` | `string` | 资源 message 类型；可使用全限定名。 |
| `version.kind` | `string` | 更新并发控制：`STRONG`、`WEAK` 或 `NONE`。 |
| `version.type` | `string` | 版本值类型：`U64`、`U32` 或 `STRING`；`STRONG`、`WEAK` 时需要。 |
| `reader` | `object` | 读取能力。配置 `reader: {}` 即生成 `Get`。 |
| `reader.batch` | `bool` | 生成 `BatchGet`。 |
| `reader.list` | `bool` | 生成 `List`，请求包含分页、`filter` 和 `order_by`。 |
| `reader.list_config.total_size` | `bool` | List 响应是否包含 `total_size`；省略或为 `true` 时包含。 |
| `reader.http` | `object` | 仅覆盖 `List` 的 HTTP `verb`、`path`、`body` 或 `body_style`。 |
| `writer.update` | `object` | 配置后生成 `Update`。 |
| `writer.update.mask` | `bool` | 是否在 Update 请求中加入 `google.protobuf.FieldMask` 类型的 `update_mask`。 |
| `writer.update.http` | `object` | 覆盖 Update 的 HTTP `verb`、`path`、`body` 或 `body_style`。 |
| `options` | `[]object` | 已解析并校验的预留配置；当前不会写入生成的 proto option。 |

#### 版本策略

| 策略 | Get 响应 | Update 请求 | Update 响应 | 使用场景 |
|---|---|---|---|---|
| `STRONG` | 返回标量 `version` | 必须携带标量 `version` 参与 CAS | 返回更新后的标量版本 | 需要强制防止并发覆盖 |
| `WEAK` | 返回 wrapper 类型 `version` | 可选携带 wrapper 类型 `version` | 返回更新后的 wrapper 版本 | 支持客户端选择是否进行 CAS |
| `NONE` | 不返回版本 | 不携带版本 | 返回 `google.protobuf.Empty` | 无需乐观锁的直接更新 |

`STRONG` 的标量类型、`WEAK` 的对应 wrapper 类型由 `version.type` 决定：`U64`、`U32` 或 `STRING`。

### `services`：服务暴露与自定义方法

一个 service 可暴露实体的全部能力，也可将资源和方法收窄到一个子集。

| 字段 | 类型 | 作用 |
|---|---|---|
| `services[].name` | `string` | Service 名，使用 `PascalCase`，如 `LibraryService`。 |
| `services[].entities[].name` | `string` | 引用已在 `entities` 中定义的实体。 |
| `services[].entities[].resources` | `[]object` | 可选的收窄规则；当前使用资源 `name`、`reader.batch`、`reader.list` 与 `writer.update`。省略时暴露该实体的全部能力。 |
| `services[].custom_methods` | `[]object` | service 级自定义 RPC 列表。 |
| `custom_methods[].name` | `string` | 自定义 RPC 名，使用 `PascalCase`。 |
| `custom_methods[].request` | `string` | Request message 类型。 |
| `custom_methods[].response` | `string` | Response message 类型。 |
| `custom_methods[].http` | `object` | 启用 HTTP 后可设置 `verb`、`path`、`body`；路径支持 AIP-136 冒号语法。 |

```yaml
services:
  - name: LibraryService
    entities:
      - name: book                 # 暴露 book 的全部能力

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

## 项目结构

```text
aip-gen/
├── cmd/apigen/        # CLI 入口
├── internal/
│   ├── cli/           # Cobra 命令
│   ├── yaml/          # api.yaml 解析与校验
│   ├── dep/           # path / Git / BSR 依赖解析
│   ├── ir/            # 中间表示与实体建模
│   ├── render/        # Proto 与 HTTP 注解渲染
│   └── build/         # Protobuf 插件编排
├── examples/book/     # 端到端示例
├── openspec/          # 设计变更记录
└── design-v2.md       # 架构设计与路线图
```

## 开源许可证

项目声明采用 **MIT License**。当前仓库尚未包含 `LICENSE` 正文；在发布、再分发或作为依赖使用前，请补充正式许可文件。
