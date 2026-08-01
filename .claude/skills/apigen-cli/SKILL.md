---
name: apigen-cli
description: >
  当用户需要操作 apigen CLI 工具时使用此 skill — apigen 是一个声明式 API 生成工具，
  通过 api.yaml 配置文件生成 AIP 风格的 gRPC 服务定义。在用户需要安装 apigen、执行
  apigen 命令（generate/build/entity list/dep update/dep prune）、编写或修改 api.yaml
  配置、排查 apigen 错误时触发。覆盖 proto 生成、gRPC/HTTP/OpenAPI/TypeScript 编译、
  依赖管理及所有配置字段。
---

# apigen CLI

声明式 API 生成工具。通过 `api.yaml` 配置生成 AIP 风格的 gRPC 服务 `.proto` 文件，
并可编译为 Go、HTTP 网关、OpenAPI v2 和 TypeScript 客户端代码。

仓库：`github.com/hellowzsg/api-gen` | Go 1.24+ | MIT

## 安装

```bash
go install github.com/hellowzsg/api-gen/cmd/apigen@latest
apigen --help
```

必需的 protoc 插件（必须在 `PATH` 中）：

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

可选插件 — 仅在启用对应功能时安装：

```bash
# HTTP + OpenAPI
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

# TypeScript
npm install --global @bufbuild/protoc-gen-es
```

环境变量：

| 变量 | 作用 | 默认值 |
| --- | --- | --- |
| `APIGEN_CACHE_DIR` | 依赖缓存根目录 | `~/.cache/apigen` |
| `APIGEN_LOG_LEVEL` | 日志级别（debug/info/warn/error）；覆盖 `-v` | `warn` |

## 命令

所有子命令支持 `-f`/`--file`（默认 `./api.yaml`）。全局标志：`-v`/`--verbose`（结构化日志输出到 stderr）。

### `apigen generate [-f api.yaml]`

校验配置 → 解析依赖 → 编译 proto → 渲染 service `.proto` 文件到 `settings.out.proto`。
使用原子写入（全部 service 成功 → 替换输出；失败则保留原有输出）。

### `apigen build [-f api.yaml]`

执行 `generate`，然后调用已安装的 protoc 插件。插件选择由配置驱动：

| 插件 | 触发条件 | 输出目录 |
| --- | --- | --- |
| `protoc-gen-go` | 始终 | `out.go` |
| `protoc-gen-go-grpc` | 始终 | `out.go` |
| `protoc-gen-grpc-gateway` | `http.enable` | `out.go` |
| `protoc-gen-openapiv2` | `http.enable` + `http.generate_openapi` + `out.openapi` 已设置 | `out.openapi` |
| `protoc-gen-es` | `plugins.js` 含 `es` + `out.js` 已设置 | `out.js` |

仅编译生成的 service proto + 本地 `import_protos.path` 声明的用户 proto。远程依赖和 WKT 仅链接，不生成代码。

### `apigen entity list [-f api.yaml]`

干跑预览。将所有实体、资源及生成的 RPC 名称打印到 stdout。不写入任何文件。使用宽松 HTTP 模式（无需 key descriptor）。

### `apigen dep update [-f api.yaml]`

强制重新拉取所有远程依赖（Git + BSR）。更新 `api.lock` 中 Git 依赖的 resolved commit。
Git 仓库缓存在 `$APIGEN_CACHE_DIR` 下，按 `URL+commit` 索引。

### `apigen dep prune [-f api.yaml]`

移除 `api.lock` 中未被引用的 Git 条目。当前为预留空操作。

## 配置工作流

### 快速开始流程

1. 编写 key type 和 resource type 的 proto message（用户维护）
2. 编写 `api.yaml` 声明实体、资源和服务
3. 执行 `apigen entity list -f api.yaml` 验证
4. 执行 `apigen generate -f api.yaml` 生成 `.proto`
5. 执行 `apigen build -f api.yaml` 编译全部产物

### 最小 api.yaml 骨架

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

### 类型引用规则

- **简写形式**（如 `BookId`）：message 所在 proto package 与 `api.yaml` 的 `name` 相同时可用；apigen 自动补全为 `<name>.<message>`
- **全限定名**（如 `demo.common.ShelfId`）：message 所在 package 与 `name` 不同时必须使用

### Create 主键模式

- `create: {}` 或 `create: { key: server }` — 服务端生成主键（请求体中不含 key）
- `create: { key: client }` — 客户端指定主键（key 位于字段 1）

发布后切换模式属于 breaking change（字段号会变化）。

### 版本策略

| 策略 | Get 响应 | Update 请求 | Update 响应 | 使用场景 |
| --- | --- | --- | --- | --- |
| `STRONG` | 标量 `version` | 标量 `version`（CAS） | 更新后的标量版本 | 强制 CAS |
| `WEAK` | wrapper `version` | 可选 wrapper `version` | 更新后的 wrapper 版本 | 可选 CAS |
| `NONE` | 无版本 | 无版本 | `google.protobuf.Empty` | 无乐观锁 |

## 方法生成映射

| 配置 | 生成的方法 | 层级 |
| --- | --- | --- |
| `create: {}` | `Create<Entity>` | 实体写 |
| `create: { batch: true }` | `Create<Entity>` + `BatchCreate<Entity>s` | 实体写 |
| `create: { key: client }` | `Create<Entity>`（请求含 key） | 实体写 |
| `delete: {}` | `Delete<Entity>` | 实体写 |
| `delete_soft: {}` | `Delete<Entity>Soft` | 实体写 |
| `reader: {}` | `Get<Entity><Resource>` | 资源读 |
| `reader.batch: true` | `BatchGet<Entity><Resource>s` | 资源读 |
| `reader.list: true` | `List<Entity><Resource>s` | 资源读 |
| `writer.update: {}` | `Update<Entity><Resource>` | 资源写 |
| `custom_methods[].name` | `<Name>` | 自定义 RPC |
| `custom_methods[].stream` | `<Name>`（流式变体） | 自定义 RPC |

`custom_methods[].stream` 取值：

| 取值 | proto 生成 | HTTP 支持 |
| --- | --- | --- |
| `""`（默认，省略） | `rpc X(Req) returns (Resp)` | 是 |
| `server` | `rpc X(Req) returns (stream Resp)` | 是（chunked 响应） |
| `client` | `rpc X(stream Req) returns (Resp)` | 否（编译期报错） |
| `bidi` | `rpc X(stream Req) returns (stream Resp)` | 否（编译期报错） |

`bidi` 是 `bidirectional` 的标准 gRPC 缩写。`stream` 字段一经发布不可切换（unary ↔ streaming 属于 breaking change）。

HTTP 路径（flat 风格，`http.enable` 时）：

| 方法 | 路径 |
| --- | --- |
| Create（服务端 key） | `POST /{prefix}/{Service}/{collection}` |
| Create（客户端 key） | `POST /{prefix}/{Service}/{collection}/{key叶子段...}` |
| Get | `GET /{prefix}/{Service}/{collection}/{key叶子段...}/{resource}` |
| BatchGet | `POST /{prefix}/{Service}/{collection}/{resource}/batchGet` body:`"*"` |
| List | `POST /{prefix}/{Service}/{collection}/{resource}/list` body:`"*"` |
| Update | `PATCH /{prefix}/{Service}/{collection}/{key叶子段...}/{resource}` |
| Delete | `DELETE /{prefix}/{Service}/{collection}/{key叶子段...}` |
| DeleteSoft | `POST /{prefix}/{Service}/{collection}/deleteSoft` body:`"*"` |

## 故障排除

常见错误及修复方法：

- **`plugin "protoc-gen-X" not found`** — 安装缺失的插件；确保 `$GOPATH/bin` 在 `PATH` 中
- **`decode api.yaml: unknown field`** — YAML 严格解析；检查拼写（注意 `type_` 带下划线）
- **`type not found` / `not a message`** — 确认包含该 message 的 proto 文件已在 `import_protos` 中声明；如果 message 在不同 package 中，使用全限定名
- **`key type descriptor not found`**（HTTP 启用时）— `import_protos` 必须包含 key message 所在的 proto 文件；`entity list` 使用宽松 HTTP 模式可绕过此检查
- **`google/api/annotations.proto` 缺失** — 启用 HTTP 时，确保 googleapis 可通过 `path`（本地 vendored）、`git` 或 `bsr` 获取
- **Git/BSR 拉取失败** — 检查网络、CLI 可用性（`git --version`、`buf --version`）；清理缓存：`rm -rf ~/.cache/apigen && apigen dep update`
- **`api.lock` 损坏** — 删除后重新执行 `apigen dep update`
- **旧输出残留 / `.bak` 目录** — 删除 `<out>.bak` 后重新生成
- **`protoc-gen-es` 排序错误** — apigen 已对 proto 文件做拓扑排序；若仍报错则表明用户 proto 存在循环 import

## 设计约束

生成或编辑 apigen 配置时需注意的关键行为：

1. **无需 protoc** — `apigen build` 通过 Protobuf 插件协议直接与插件通信
2. **原子写入** — `generate` 先写入 `.apigen-stage-*` 临时目录，成功后原子替换输出目录
3. **严格 YAML** — `api.yaml` 中未知字段直接报错
4. **类型引用规则** — 简写形式仅在 message package 等于 `name` 时可用；否则必须全限定名
5. **Create key 模式不可切换** — `server` ↔ `client` 切换会改变字段号，属于 breaking change
6. **拓扑排序** — proto 文件在传递给插件前按依赖排序，以兼容 `protoc-gen-es`
7. **并行插件编译** — 所有插件通过 errgroup 并发执行，共享只读 CodeGeneratorRequest
8. **缓存版本化** — 依赖缓存在 `~/.cache/apigen/v1/` 下；升级 `dep.CacheVersion` 可失效旧缓存
9. **日志输出到 stderr** — stdout 保留给命令结果（`entity list`），不被日志污染
10. **api.lock 仅记录 Git** — BSR 依赖不写入 lock 文件（由 buf 自行管理）

## 参考文件

详细的配置 schema、字段表、完整使用示例和项目结构，参见：

- `references/config-schema.md` — 完整的 `api.yaml` 字段参考，包含全部实体、资源、服务和 settings 字段
- `references/examples.md` — 8 个使用场景：最小 P0、全量 HTTP、跨 package 引用、客户端 key、Service 收窄、远程依赖、干跑预览、源码调试
