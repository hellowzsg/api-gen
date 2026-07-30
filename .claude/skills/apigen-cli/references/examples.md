# 使用示例

## 场景 1：最小 P0 — 纯 gRPC（仅 Get + Update）

```yaml
syntax: v1
name: simple.config

import_protos:
  - path: "proto/**/*.proto"

settings:
  go_repo: github.com/acme/simple-config
  out:
    proto: generated/proto
    go: generated/go

entities:
  - name: config
    key: { type_: ConfigId }
    resources:
      - name: entry
        type_: ConfigEntry
        version: { kind: NONE }
        reader: {}
        writer:
          update: {}

services:
  - name: ConfigService
    entities:
      - name: config
```

**生成方法**：`GetConfigEntry`、`UpdateConfigEntry`（返回 `google.protobuf.Empty`）

**命令**（仅需 `protoc-gen-go` + `protoc-gen-go-grpc`）：
```bash
apigen generate -f api.yaml
apigen build -f api.yaml
```

---

## 场景 2：全量特性 — HTTP + OpenAPI + TypeScript + custom_methods

```yaml
syntax: v1
name: demo.business.book

import_protos:
  - path: "proto/**/*.proto"
  - git: https://github.com/googleapis/googleapis
    ref: master

settings:
  go_repo: github.com/acme/demo-book
  js_repo: "@acme/demo-book"
  out:
    proto: generated/proto
    go: generated/go
    js: generated/js
    openapi: generated/openapi
  http:
    enable: true
    prefix: /library
    generate_openapi: true
  plugins:
    js: [es]

entities:
  - name: book
    key: { type_: BookId }
    create: {}
    delete: {}
    delete_soft: {}
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: STRONG, type: U64 }
        reader:
          batch: true
          list: true
          list_config:
            total_size: true
            filter_type: BookMetaFilter
        writer:
          update: { mask: true }
      - name: content
        type_: BookContent
        version: { kind: NONE }
        reader: {}
        writer:
          update: {}

services:
  - name: LibraryService
    entities:
      - name: book
    custom_methods:
      - name: ArchiveBook
        request: ArchiveBookRequest
        response: ArchiveBookResponse
        http:
          verb: post
          path: /library/LibraryService/book/{book_id}:archive
          body: "*"
```

**命令**（需要全部插件）：
```bash
apigen generate -f api.yaml
apigen build -f api.yaml
cd examples/book && go test -v -count=1 ./...
```

---

## 场景 3：跨 Package 类型引用

当 key/resource message 所在 package 与 `api.yaml` 的 `name` 不同时：

```yaml
name: demo.business.book

entities:
  - name: shelf
    key: { type_: demo.common.ShelfId }
    resources:
      - name: meta
        type_: demo.common.Shelf
        version: { kind: NONE }
        reader: {}
        writer:
          update: {}
```

---

## 场景 4：客户端指定主键

```yaml
entities:
  - name: note
    key: { type_: NoteId }
    create: { key: client }
    resources:
      - name: meta
        type_: NoteMeta
        version: { kind: NONE }
        reader: {}
        writer:
          update: {}
```

生成的 `CreateNote` 请求：`{ NoteId key = 1; NoteMeta meta = 2; }`
HTTP 路径：`POST /{prefix}/{Service}/note/{key.id}` body:`"*"`

---

## 场景 5：Service 收窄

```yaml
services:
  - name: LibraryService
    entities:
      - name: book          # 暴露全部能力

  - name: AdminService
    entities:
      - name: book
        resources:
          - name: meta
            reader: { list: true }  # 仅暴露 ListBookMetas
```

---

## 场景 6：远程依赖

```yaml
import_protos:
  - path: "proto/**/*.proto"
  - git: https://github.com/googleapis/googleapis
    ref: master
    subdir: google
  - bsr: buf.build/googleapis/googleapis
```

```bash
apigen dep update -f api.yaml  # 拉取依赖并写入 api.lock
apigen build -f api.yaml        # 使用锁定版本构建
```

---

## 场景 7：干跑预览

```bash
apigen entity list -f api.yaml          # 预览生成内容，不写文件
apigen entity list -f api.yaml -v       # 含 verbose 依赖日志
```

---

## 场景 8：源码级开发

```bash
cd <project-root>

go run ./cmd/apigen entity list -f examples/book/api.yaml
go run ./cmd/apigen generate -f examples/book/api.yaml
go run ./cmd/apigen build -f examples/book/api.yaml

go test ./internal/... -v
cd examples/book && go test -v -count=1 ./...
```
