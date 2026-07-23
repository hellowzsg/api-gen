## Why

项目当前有 30 个 `*_test.go` 单元测试文件分布在 `internal/` 各子包中，以及 `examples/book/` 下的 4 个 e2e 测试文件。但现有 e2e 测试仅覆盖 book 示例的正向路径，缺乏：

1. **系统化的反向测试** — 错误输入、异常路径、fail-fast 校验未覆盖
2. **边界条件测试** — 嵌套 key、多资源、custom_method 等边界场景
3. **统一的测试组织** — 测试分散在各模块，缺少专属目录系统化管理
4. **完整的错误路径覆盖** — YAML 解析、类型引用、Service 引用、HTTP 配置、IR 构建、key leaves、option 注入等 13 层错误路径未系统覆盖

## What Changes

1. **新建 `testcase/` 目录** — 作为独立 Go module，系统化组织所有 e2e 测试
2. **正向测试套件（`positive/`）** — 覆盖 generate 产物验证、gRPC 全方法、HTTP 全路由、OpenAPI spec、simple P0 纯 gRPC 验证、edge 边界条件验证
3. **反向测试套件（`negative/`）** — 覆盖 13 层错误路径共 ~50 个错误场景，外加依赖解析层 3 个错误路径（protocompile 失败、api.lock 损坏、glob 无匹配）
4. **fixtures 驱动** — book/simple/edge/invalid 四类测试输入，invalid 含 ~43 个错误配置文件
5. **CI 集成** — 新增 `testcase-e2e` CI job

## Impact

- **新增目录**：`testcase/`（含 go.mod、fixtures、positive、negative）
- **CI 配置**：`.github/workflows/ci.yml` 新增 `testcase-e2e` job
- **.gitignore**：补充 `testcase/fixtures/*/generated/`
- **现有代码**：不修改任何 `internal/` 或 `examples/` 代码
- **依赖**：testcase module 引用 grpc、grpc-gateway、protobuf 等库
