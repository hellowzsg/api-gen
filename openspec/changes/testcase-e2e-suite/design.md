## Context

aip-gen 是一个 API 代码生成工具，从 YAML 配置（`api.yaml`）生成 proto 定义、Go gRPC stub、HTTP gateway、OpenAPI spec、TypeScript stub。项目已有 30 个 `*_test.go` 单元测试和 `examples/book/` 下的 4 个 e2e 测试，但缺乏系统化的反向测试和边界条件覆盖。

现有 e2e 测试模式（`examples/book/e2e_grpc_test.go`、`e2e_http_test.go`）采用：
- in-memory `bufconn` gRPC server + mock server
- `httptest.Server` + `runtime.ServeMux` HTTP gateway
- 生成产物作为独立 Go module（`github.com/acme/demo-book`）

## Goals / Non-Goals

**Goals:**
- 在 `testcase/` 目录下建立系统化的 e2e 测试体系
- 正向测试覆盖 generate 产物、gRPC 全方法、HTTP 全路由、OpenAPI spec
- 反向测试覆盖 13 层错误路径共 ~50 个错误场景
- fixtures 驱动的测试组织
- CI 集成自动运行

**Non-Goals:**
- 不做分支覆盖率统计（e2e 覆盖率对 internal 意义有限，internal 已有 30 个单元测试）
- 不修改现有 `internal/` 或 `examples/` 代码
- 不迁移现有 `*_test.go` 文件（保持 Go 同包测试惯例）
- 不覆盖 dep 层的 git/bsr fetch 失败（需要网络 mock，成本高收益低）
- 不覆盖编译层的 protoc 插件缺失（CI 环境已保证插件存在）

## Decisions

### D1: testcase 作为独立 Go module

`testcase/go.mod` 声明为 `module github.com/hellowzsg/api-gen/testcase`，引用生成的 stub module（类似 `examples/book/go.mod` 引用 `github.com/acme/demo-book`）。

理由：与现有 `examples/book/` 模式一致，e2e 测试需要 import 生成的 proto stub，独立 module 最自然。

### D2: fixtures 中的生成产物 gitignored

`testcase/fixtures/*/generated/` 加入 `.gitignore`，由 CI 和本地 TestMain 自动生成。

理由：生成产物是派生物，不应纳入版本控制。

### D3: 反向测试直接调用 apigen generate

反向测试通过构建 apigen 二进制或直接调用 `internal/cli` 包的导出函数，对 `fixtures/invalid/*.yaml` 运行 generate 并断言错误。

理由：反向测试验证的是"生成过程应失败"，不需要预先生成产物。

### D4: 正向测试 mock 复用现有模式

从 `examples/book/e2e_http_test.go` 复制 mock server 和 helper（`newGRPCServer`、`newGatewayMux`、`doReq`、`mustReadJSON`），放在 `testcase/positive/helpers_test.go` 中并扩展。

理由：避免重复编写 mock 基础设施，保持与现有 e2e 测试一致的风格。

### D5: 不做覆盖率统计

e2e 测试运行在独立 module 中，跨 module 覆盖率（`-coverpkg`）支持不佳且意义有限。`internal/` 已有 30 个单元测试覆盖分支逻辑。

### D6: skip_preci = true

根据项目约定，aip-gen 项目不执行 PreCI 代码规范检查。

### D7: 每个 fixture 使用独立 go.mod（实现时决策）

实现时发现 testcase 主 module 无法直接引用 `fixtures/*/generated/go/...` 下的生成代码，因为生成代码的 import path 包含 fixture module 路径。解决方案：每个 fixture（book/simple/edge）创建独立的 `go.mod`，在 testcase 主 module 的 `go.mod` 中用 `replace` 指令指向本地路径。

这与 `examples/book/go.mod` 的模式一致。

### D8: proto go_package 与输出路径对齐（实现时决策）

实现时发现 protoc 的输出路径遵循 proto 源文件路径（如 `proto/demo/business/book/book.proto` → `generated/go/demo/business/book/`），而非 go_package 的子路径。对于 simple/edge fixture（proto 文件在根目录），go_package 必须设为 `generated/go`（而非 `generated/go/simple/config`），否则 import path 与实际文件路径不匹配导致编译失败。

### D9: http_without_googleapis fixture 使用非通配路径（实现时决策）

`hasGoogleapisDependency()` 对包含 `**` 的 glob 路径保守返回 true（认为可能覆盖 vendored google/api 目录）。因此 `http_without_googleapis.yaml` 必须使用非通配路径（如 `proto/specific.proto`）才能正确触发 "no googleapis dependency" 错误。

### D10: key_type_empty 触发 "type_ is required" 而非 "type_ is empty"（实现时决策）

YAML 解析层的 `validate()` 方法在 `ValidateReferences()` 之前运行，检查 `e.Key.Type == ""` 并返回 "type_ is required"。空字符串 `type_: ""` 被解析为空值，因此触发的是 YAML 层的校验错误而非类型引用层的错误。测试断言相应调整为 "type_ is required"。

### D11: edge fixture 跨 package 类型引用（补充场景）

新增跨 package 类型引用验证场景：edge fixture 中 doc entity 的类型（`DocId`、`DocMeta`、`DocContent`、`DocLog`）来自 `edge.example` package，tag entity 的类型（`TagId`、`Tag`）来自 `edge.tag` package。同一 `api.yaml` 同时引用两个不同 proto package 的 key/resource types。

- 在 `api.yaml` 中，跨 package 的类型必须使用全限定名（如 `edge.tag.TagId`），简写形式会被解析为 api.yaml `name` 对应的默认 package（`edge.example`）。
- 生成的 service proto 正确 import 了 `edge.proto` 和 `edge/tag/tag.proto` 两个文件。
- 测试覆盖：验证各 entity 的 request/response message 使用正确的 package 前缀，不出现跨 entity 的类型污染。

## Risks / Trade-offs

- **风险：fixture 维护成本** — invalid/ 下 ~41 个 YAML 文件需要维护。缓解：每个文件极简（仅触发一个错误），注释标明对应错误码。
- **风险：TestMain 生成产物较慢** — 每次测试前需运行 generate/build。缓解：CI 中有显式生成步骤，本地 TestMain 做增量检测（产物存在则跳过）。
- **权衡：独立 module 增加 go.mod 维护** — 需要维护 testcase + 3 个 fixture 的依赖版本。缓解：依赖版本与 examples/book 保持一致。
- **风险：多 module replace 链** — testcase → fixtures/book/simple/edge 三层 replace，新增 fixture 需同步更新 go.mod。缓解：README 中有新增 fixture 指南。
