# apigen examples

本目录包含 apigen 的使用示例。

## book 示例

一个图书管理服务，展示 apigen P0（纯 gRPC）的全部能力。

### 文件结构

```
book/
├── api.yaml                                    # 四段式配置（用户编写）
├── proto/
│   └── demo/business/book/
│       └── book.proto                          # 用户手写 type_（BookId/BookMeta/BookContent）
└── generated/                                  # apigen 生成（期望输出）
    └── proto/
        ├── library_service/
        │   └── library_service.proto           # LibraryService 全量方法
        └── admin_service/
            └── admin_service.proto             # AdminService 收窄（仅 ListBookMetas）
```

### 覆盖的特性

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

### 运行

```bash
# 生成 proto
apigen generate -f examples/book/api.yaml

# 生成 + 编译成 Go stub（需要 protoc-gen-go / protoc-gen-go-grpc）
apigen build -f examples/book/api.yaml

# 预览实体/资源/方法清单
apigen entity list -f examples/book/api.yaml
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
2. 编写 `api.yaml` 声明实体/资源/读写策略
3. 运行 `apigen generate -f api.yaml`
