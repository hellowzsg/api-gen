# apigen examples

本目录包含 apigen 的使用示例。两个示例各有明确侧重点，不重复：

| 示例 | 目录 | 定位 | HTTP | OpenAPI | JS |
|------|------|------|:----:|:-------:|:--:|
| **simple** | `simple/` | 最小 P0 骨架：Get + Update only | ❌ | ❌ | ❌ |
| **book** | `book/` | 全量特性：P0 + P1 + P2 全覆盖 | ✅ | ✅ | ✅ |

---

## simple 示例

最小可用的 P0 纯 gRPC 配置。只生成 `Get` + `Update` 两个 RPC，展示 `api.yaml` 的最小骨架。

### 为什么需要这个示例？

- **入门起点**：30 秒理解 apigen 的最小配置结构
- **不重复 book**：book 有 Create/Delete/DeleteSoft/BatchGet/List/HTTP/OpenAPI/JS/custom_method，simple 只有 Get+Update
- **版本 NONE**：展示无乐观锁的 Update（返回 `google.protobuf.Empty`，无 mask）

### 文件结构

```
simple/
├── api.yaml                    # 最小配置（15 行核心声明）
├── proto/
│   └── config.proto            # ConfigId + ConfigEntry
└── generated/                  # apigen 生成
    └── proto/
        └── config_service/
            └── config_service.proto
```

### 配置详解

```yaml
entities:
  - name: config
    key: { type_: ConfigId }
    # 无 create / delete / delete_soft — 不生成实体级写方法
    resources:
      - name: entry
        type_: ConfigEntry
        version: { kind: NONE }      # 无乐观锁
        reader: {}                    # 只生成 Get（无 batch/list）
        writer:
          update: {}                  # 生成 Update（无 mask）
```

### 生成的 RPC

```
service ConfigService {
  rpc GetConfigEntry(GetConfigEntryRequest) returns (GetConfigEntryResponse);
  rpc UpdateConfigEntry(UpdateConfigEntryRequest) returns (google.protobuf.Empty);
}
```

### 运行

```bash
# 预览
apigen entity list -f examples/simple/api.yaml

# 生成 proto
apigen generate -f examples/simple/api.yaml

# 生成 + 编译 Go stub（仅需 protoc-gen-go / protoc-gen-go-grpc）
apigen build -f examples/simple/api.yaml
```

---

## book 示例

全量特性展示，覆盖 P0（纯 gRPC）+ P1（HTTP MVP）+ P2（HTTP 增强）。

### 文件结构

```
book/
├── api.yaml                                    # 四段式配置（含 settings.http + custom_methods）
├── api.lock                                    # Git 依赖锁文件
├── go.mod / go.sum                             # 独立 Go module（用于 e2e 测试）
├── e2e_http_test.go                            # HTTP 端到端测试
├── e2e_openapi_test.go                         # OpenAPI 规范验证测试
├── jstub_test.go                               # TypeScript stub 生成测试
├── proto/
│   ├── demo/business/book/
│   │   └── book.proto                          # 用户手写 type_
│   ├── demo/common/
│   │   └── types.proto                         # 跨 package 共享类型
│   └── google/api/                             # vendored googleapis
└── generated/
    ├── proto/
    │   ├── library_service/library_service.proto
    │   └── admin_service/admin_service.proto
    ├── go/  (*.pb.go / *_grpc.pb.go / *.pb.gw.go)
    ├── js/  (*_pb.ts)
    └── openapi/  (*.swagger.json)
```

### 覆盖的特性

#### P0（纯 gRPC）

| 特性 | 示例中的体现 |
|------|-------------|
| 实体级 Create（只返回 key） | `CreateBook` → `CreateBookResponse { BookId key = 1; }` |
| 实体级 Delete + DeleteSoft 共存 | `DeleteBook`（硬删）+ `DeleteBookSoft`（软删） |
| 资源级 Get（STRONG 回带 version） | `GetBookMeta` → `Response { BookMeta=1; uint64 version=2; }` |
| 资源级 BatchGet | `BatchGetBookMetas`（`repeated keys` → `repeated metas`） |
| 资源级 List（分页/过滤/排序/total_size） | `ListBookMetas` |
| 资源级 Update（STRONG CAS + mask） | `UpdateBookMeta`（meta/key/update_mask/version → Response{version}） |
| 资源级 Update（NONE，返 Empty） | `UpdateBookContent`（content/key/update_mask → Empty） |
| 多 service 复用实体 + 收窄 | `AdminService` 只暴露 `ListBookMetas` |
| api-linter 豁免 | proto 顶部行内注释 |
| type_ 全限定名 + 跨 package 引用 | `demo.business.book.BookMeta` 引用 `demo.common.Timestamp` |

#### P1（HTTP MVP via grpc-gateway）

| 特性 | 示例中的体现 |
|------|-------------|
| `settings.http.enable` + `prefix` | `http: { enable: true, prefix: /library }` |
| flat 风格 HTTP 路径 | `/{prefix}/{service}/{collection}/{key叶子段...}/{resource}` |
| key 标量叶子做 URL path 段 | `BookId.id` → `{key.id}` |
| grpc-gateway 生成 `*.pb.gw.go` | `library_service.pb.gw.go` |
| BatchGet/List 用 POST + `body:"*"` | `POST /book/meta/batchGet` |
| DeleteSoft 用 POST + `body:"*"` | `POST /book/deleteSoft` |
| `google.api.http` 注解 | 每个 RPC 上的 `option (google.api.http) = {...}` |
| 多 service 独立 HTTP 路由前缀 | `/library/LibraryService/...` vs `/library/AdminService/...` |

#### P2（HTTP 增强）

| 特性 | 示例中的体现 |
|------|-------------|
| OpenAPI v2 生成 | `generated/openapi/*.swagger.json` |
| 逐方法 http 覆盖 | `reader.http` 将 List 覆盖为 GET |
| custom_methods HTTP 路由（AIP-136） | `POST /library/LibraryService/book/{book_id}:archive` |
| TypeScript stub 生成 | `plugins.js: [es]` → `generated/js/*_pb.ts` |

### 运行

```bash
# 生成 proto
apigen generate -f examples/book/api.yaml

# 生成 + 编译（需要全部插件）
apigen build -f examples/book/api.yaml

# 端到端测试
cd examples/book && go test -v -count=1 ./...
```

### 端到端测试

| 测试文件 | 验证内容 |
|---------|---------|
| `e2e_http_test.go` | 所有 HTTP 端点的路由匹配、path 变量解析、body 解码、响应序列化、404、custom_method 路由 |
| `e2e_openapi_test.go` | swagger.json 有效 JSON + 包含 custom_method 和 overridden List 路径 |
| `jstub_test.go` | protoc-gen-es 触发 + TS 文件包含 service 定义和 import/export |

---

## CI 集成

`.github/workflows/ci.yml` 在每次 push/PR 时运行 4 个并行 job：

| Job | 内容 |
|-----|------|
| **Unit Tests** | `internal/...` 全部单元测试 + `go vet` |
| **Example Generate** | 验证 simple + book 的 `apigen generate` 生成正确 proto |
| **Example Build (book)** | 安装全部插件 → `apigen build` → 验证 Go/HTTP/OpenAPI 产物 → 运行 e2e 测试 |
| **Example Build (simple)** | P0 构建 → 验证 Go stub → 确认无 gateway 文件 |

---

## 自定义示例

1. 编写 proto 定义你的 type_（key + resource message）
2. 编写 `api.yaml` 声明实体/资源/读写策略
3. 若启用 HTTP，确保 `import_protos` 包含 googleapis
4. 运行 `apigen generate -f api.yaml`（或 `apigen build` 同时编译）

### 选择哪个示例作为起点？

| 场景 | 起点 |
|------|------|
| 快速理解最小配置 | `simple/` |
| 需要完整 HTTP + OpenAPI + JS | `book/` |
