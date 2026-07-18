# apigen examples

本目录包含 apigen 的使用示例。

## book 示例

一个图书管理服务，展示 apigen P0（纯 gRPC）+ P1（HTTP MVP via grpc-gateway）的全部能力。

### 文件结构

```
book/
├── api.yaml                                    # 四段式配置（用户编写，含 settings.http）
├── proto/
│   ├── demo/business/book/
│   │   └── book.proto                          # 用户手写 type_（BookId/BookMeta/BookContent）
│   └── google/api/                             # vendored googleapis（annotations.proto + http.proto）
└── generated/                                  # apigen 生成（期望输出）
    ├── proto/
    │   ├── library_service/
    │   │   └── library_service.proto           # LibraryService 全量方法（含 google.api.http 注解）
    │   └── admin_service/
    │       └── admin_service.proto             # AdminService 收窄（含独立 HTTP 路由前缀）
    └── go/
        ├── library_service/
        │   ├── library_service.pb.go
        │   ├── library_service_grpc.pb.go
        │   └── library_service.pb.gw.go        # grpc-gateway 反向代理（P1）
        ├── admin_service/
        │   └── ...（同上）
        └── demo/business/book/
            └── book.pb.go
```

### 覆盖的特性

#### P0（纯 gRPC）

| 特性 | 示例中的体现 |
|------|-------------|
| 实体级 Create（各资源可选，只返回 key） | `CreateBook` → `CreateBookResponse { BookId key = 1; }` |
| 实体级 Delete + DeleteSoft 共存 | `DeleteBook`（硬删）+ `DeleteBookSoft`（软删） |
| 资源级 Get（STRONG 回带 version） | `GetBookMeta` → `GetBookMetaResponse { BookMeta=1; uint64 version=2; }` |
| 资源级 BatchGet | `BatchGetBookMetas`（`repeated keys` → `repeated metas`） |
| 资源级 List（分页/过滤/排序/total_size） | `ListBookMetas`（page_size/page_token/filter/order_by + next_page_token/total_size） |
| 资源级 Update（STRONG CAS + mask） | `UpdateBookMeta`（meta=1, key=2, update_mask=3, version=4 → Response{version=1}） |
| 资源级 Update（NONE，无 version，返 Empty） | `UpdateBookContent`（content=1, key=2, update_mask=3 → Empty） |
| 多 service 复用实体 + 收窄 | `AdminService` 只暴露 `ListBookMetas` |
| api-linter 豁免（按实际触发裁剪） | proto 顶部行内注释 |
| filter/order_by 统一 string | `ListBookMetasRequest.filter=3, order_by=4`（类型 string） |
| type_ 全限定名引用 | `demo.business.book.BookMeta`（跨 package） |

#### P1（HTTP MVP via grpc-gateway）

| 特性 | 示例中的体现 |
|------|-------------|
| `settings.http.enable` + `prefix` | `api.yaml` 中 `http: { enable: true, prefix: /library }` |
| flat 风格 HTTP 路径 | `/{prefix}/{service}/{collection}/{key叶子段...}/{resource}` |
| key 类型递归解析标量叶子做 URL path 段绑定 | `BookId.id` → `/{key.id}`（DeleteBook/DeleteBookSoft/Get/Update 路径） |
| grpc-gateway 生成 `*.pb.gw.go` | `library_service.pb.gw.go`、`admin_service.pb.gw.go` |
| BatchGet/List 用 POST + `body:"*"` | `POST /book/meta/batchGet`、`POST /book/meta/list` |
| DeleteSoft 用 POST + `body:"*"` | `POST /book/deleteSoft`（key 走 body） |
| `google.api.http` 注解生成 | 每个 RPC 上的 `option (google.api.http) = {...}` |
| HTTP 启用时校验已有 googleapis | `proto/google/api/{annotations,http}.proto` vendored |
| 多 service 独立 HTTP 路由前缀 | LibraryService 走 `/library/LibraryService/...`，AdminService 走 `/library/AdminService/...` |
| api-linter HTTP 豁免 | `core::0133/0231/0132/0135` 的 `http-body`/`http-method` 行内注释 |

### 运行

```bash
# 生成 proto
apigen generate -f examples/book/api.yaml

# 生成 + 编译成 Go stub（需要 protoc-gen-go / protoc-gen-go-grpc / protoc-gen-grpc-gateway）
apigen build -f examples/book/api.yaml

# 预览实体/资源/方法清单
apigen entity list -f examples/book/api.yaml
```

### 端到端 HTTP 测试

`examples/book/e2e_http_test.go` 验证 P1 HTTP MVP 的端到端行为：

- 用 `runtime.NewServeMux()` + `RegisterXxxHandlerServer` 把 mock gRPC server 挂成 HTTP 反向代理
- 对每个 HTTP 端点发真实 HTTP 请求，验证：
  - 路由匹配（POST/GET/DELETE/PATCH 动词 + 路径模板）
  - key 路径变量解析（`{key.id}` → 请求字段 `Key.Id`，DELETE 无 body 也能填充）
  - body 解码（`body:"*"` 把 JSON body 反序列化到请求消息）
  - 响应序列化（uint64 → 字符串、FieldMask → 逗号分隔字符串，protojson 规范）
  - AdminService 收窄路由（独立路径前缀，不与 LibraryService 冲突）
  - 未注册路径返回 404

```bash
cd examples/book && go test -v -count=1 ./...
```

### 预期 entity list 输出

```
Entity: book (Pascal: Book, Key: demo.business.book.BookId)
  Create: CreateBook
  Delete: DeleteBook
  DeleteSoft: DeleteBookSoft
  Resource: meta (Type: demo.business.book.BookMeta, Version: STRONG)
    Get: GetBookMeta
    BatchGet: BatchGetBookMetas
    List: ListBookMetas
    Update: UpdateBookMeta
  Resource: content (Type: demo.business.book.BookContent, Version: NONE)
    Get: GetBookContent
    Update: UpdateBookContent
```

## 自定义示例

参考 `book/api.yaml` 和 `book/proto/` 创建自己的示例：

1. 编写 proto 定义你的 type_（key + resource message）
2. 编写 `api.yaml` 声明实体/资源/读写策略（按需开启 `settings.http`）
3. 若启用 HTTP，确保 `import_protos` 中包含 googleapis（或 vendored `google/api/*.proto`）
4. 运行 `apigen generate -f api.yaml`（或 `apigen build` 同时编译 Go stub）
