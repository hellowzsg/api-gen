# testcase — aip-gen 端到端测试套件

本目录是 aip-gen 的端到端 (e2e) 测试体系，作为独立 Go module 运行。

## 目录结构

```
testcase/
├── go.mod                    # testcase module (github.com/hellowzsg/api-gen/testcase)
├── fixtures/                  # 测试输入
│   ├── book/                  # 完整 HTTP + OpenAPI fixture
│   │   ├── go.mod             # book fixture module
│   │   ├── api.yaml
│   │   └── proto/
│   ├── simple/                # P0 纯 gRPC fixture（无 HTTP）
│   │   ├── go.mod
│   │   ├── api.yaml
│   │   └── proto/
│   ├── edge/                  # 边界条件 fixture（多 entity、嵌套 key、WEAK version）
│   │   ├── go.mod
│   │   ├── api.yaml
│   │   └── proto/
│   └── invalid/               # 反向测试 fixtures
│       ├── proto/             # 共享 proto 文件（含各种测试用 message 定义）
│       ├── *.yaml             # ~41 个错误配置文件
│       ├── protocompile_error/# 含语法错误的 proto 文件
│       ├── corrupt_lock/      # 损坏的 api.lock 文件
│       └── empty_dir/         # 空目录（glob 无匹配测试）
├── positive/                  # 正向 e2e 测试
│   ├── helpers_test.go        # Mock server + helper 函数
│   ├── generate_test.go       # generate 产物验证
│   ├── grpc_test.go           # gRPC 全方法测试
│   ├── http_test.go           # HTTP gateway 全路由测试
│   ├── openapi_test.go        # OpenAPI spec 验证
│   ├── simple_grpc_test.go    # simple fixture P0 纯 gRPC 验证
│   └── edge_test.go           # edge fixture 边界条件验证
└── negative/                  # 反向 e2e 测试
    ├── generate_errors_test.go # YAML/类型引用/Service引用/HTTP/路径/插件 错误
    ├── ir_errors_test.go       # key leaves/option/CLI 类型引用 错误
    ├── http_negative_test.go   # HTTP 运行时反向（404、错误 method、无效 JSON）
    ├── grpc_negative_test.go   # gRPC 运行时反向（Unimplemented、nil、cancelled）
    └── dep_errors_test.go      # 依赖解析错误（glob 无匹配、protocompile 失败、api.lock 损坏）
```

## 运行方式

```bash
# 1. 编译 apigen
cd /path/to/aip-gen
go build -o /tmp/apigen ./cmd/apigen

# 2. 生成 fixtures 产物
/tmp/apigen generate -f testcase/fixtures/book/api.yaml
/tmp/apigen build -f testcase/fixtures/book/api.yaml
/tmp/apigen generate -f testcase/fixtures/simple/api.yaml
/tmp/apigen build -f testcase/fixtures/simple/api.yaml
/tmp/apigen generate -f testcase/fixtures/edge/api.yaml
/tmp/apigen build -f testcase/fixtures/edge/api.yaml

# 3. 运行所有测试
cd testcase
go mod tidy
go test ./... -v -count=1

# 或只运行正向/反向测试
go test ./positive/... -v -count=1
go test ./negative/... -v -count=1
```

## 测试覆盖

### 正向测试（positive/）
- **generate 产物验证**：检查生成的 proto 文件结构（service、RPC、message 字段、AdminService 收窄）
- **gRPC 全方法**：LibraryService 10 个 RPC + AdminService 6 个收窄方法 + shelf 跨 package 类型引用
- **HTTP gateway 全路由**：所有 HTTP 端点（POST/GET/PATCH/DELETE + custom method 冒号语法 + shelf 路由）
- **OpenAPI spec**：swagger.json 有效性、路径完整性、HTTP verb 正确性
- **simple fixture P0**：验证无 HTTP 注解、无 annotations import、仅 gRPC service
- **edge fixture**：多 entity RPC、嵌套 key 路径绑定、WEAK/NONE/STRONG version、**跨 package 类型引用**（doc entity 的 key 在 `edge.example`，tag entity 的 key 在 `edge.tag`）

### 跨 package 类型引用

当 entity 的 `key.type_` 或 resource `type_` 所在的 proto package 与 `api.yaml` 的 `name` 不同时，必须使用**全限定名**：

- `edge` fixture：`name: edge.example`，tag entity 的 key 在 `edge.tag` package → `type_: edge.tag.TagId`
- `examples/book`：`name: demo.business.book`，shelf entity 的 key 在 `demo.common` package → `type_: demo.common.ShelfId`

详细规则见项目根 `README.md` 的"类型引用规则"章节。

### 反向测试（negative/）
覆盖 13 层错误路径 + 依赖解析层 3 个错误：
- **YAML 解析层**（A1-A12）：缺失字段、非法格式、未知字段
- **类型引用**（B1-B5）：空值、非法前缀、非法字符
- **Service 引用**（C1-C3）：重复 entity、引用不存在 entity/resource
- **HTTP 配置**（D1-D2）：无 googleapis 依赖、非法 body_style
- **路径语法**（E1-E3）：空变量、前导/尾随 dot
- **插件**（F1）：未知 JS 插件
- **key leaves**（G3-G6）：repeated/map/oneof 字段、循环引用
- **option**（I1-I7）：非法 target、缺失 path、空名、空格、非法字符
- **CLI 类型引用**（J1-J2）：key/resource 类型不在 proto 中
- **依赖解析**（K1-K3）：glob 无匹配、protocompile 失败、api.lock 损坏
- **HTTP 运行时**：404 未注册路由、错误 method、无效 JSON body
- **gRPC 运行时**：Unimplemented、nil 请求、cancelled context

## 新增 fixture 指南

### 正向 fixture
1. 在 `fixtures/<name>/` 下创建 `api.yaml` 和 `proto/` 目录
2. 创建 `go.mod`（module path = `github.com/hellowzsg/api-gen/testcase/fixtures/<name>`）
3. 在 testcase 的 `go.mod` 中添加 replace 指令
4. 在 `positive/` 下编写测试

### 反向 fixture
1. 在 `fixtures/invalid/` 下创建 `<error_name>.yaml`
2. 如需特定 proto 类型，在 `fixtures/invalid/proto/types.proto` 中添加 message 定义
3. fixture 的 `name` 应为 `test.invalid`（与共享 proto 的 package 一致）
4. 在 `negative/` 下编写 table-driven 测试，断言错误消息

## CI 集成

CI 中 `testcase-e2e` job 会自动：
1. 安装 protoc 插件
2. 编译 apigen
3. 生成所有 fixture 产物
4. 运行正向和反向测试
