## Context

微服务基于 gRPC 构建，需统一 proto 接口定义规范。现有工具（buf、protoc）解决编译/兼容性问题，但不解决服务层样板生成问题。

`design-v2.md` 定义了 apigen 工具的完整设计方案（1078行），经过多轮技术评审已定稿。本提案覆盖 P0 范围（纯 gRPC），依赖管理三路径全部纳入，option 注入完整五档，后置校验（api-linter + buf breaking）全部纳入。

### 现状

- 项目从零开始，仅有 `design-v2.md` 设计文档和空 `README.md`
- 无任何代码、无 openspec 目录
- Git 仓库在 `main` 分支

## Goals / Non-Goals

**Goals:**

1. 从四段式 `api.yaml` 生成 AIP 风格的服务层 proto（gRPC）
2. 支持实体级（Create/Delete/DeleteSoft）+ 资源级（Get/BatchGet/List/Update）方法生成
3. 支持三路径依赖管理（path/git/bsr）+ 双锁机制
4. 支持版本/乐观锁策略（STRONG/WEAK/NONE，WEAK 用 wrapper 类型）
5. 支持通用 option 注入（五档纯搬运）
6. 编译成 `*.pb.go` / `*_grpc.pb.go`（subprocess 调用 protoc-gen-*）
7. 可编译性形式化保证（四道防线）
8. 确定性输出（bit-identical）
9. api-linter 豁免 + buf breaking 后置校验

**Non-Goals:**

1. HTTP 转码（`google.api.http` 注解 + grpc-gateway）→ 后续提案（P1）
2. OpenAPI/swagger 生成 → 后续提案（P2）
3. JS stub 生成（es/connect-es）→ 后续提案
4. einride/aip-go 插件 → 后续提案
5. 逐方法 HTTP 覆盖 / body_style: resource → 后续提案（P2）
6. LRO（AIP-151）/ request_id 幂等（AIP-155）/ 变更通知机制 → 长期 Roadmap
7. 软删除增强（AIP-164 Undelete / List 排除已软删除）→ 长期 Roadmap

## Decisions

### D1: protocompile 作为 library 引用

`bufbuild/protocompile` 作为 Go library 直接引用（非 subprocess 调用 protoc）。公开 API 稳定（`Compiler`/`Resolver`/`CompositeResolver`/`WithStandardImports`），通过 `go.mod` 锁定版本 + CI 升级测试兜底。

**理由**：零外部依赖，无需 protoc 二进制；Buf CLI 自身基于此构建；核心接口已被广泛依赖。

### D2: go-git library 拉取 git 依赖

`go-git/go-git` 是纯 Go git 实现，作为 library clone 远程仓库，无需安装 git CLI。

**理由**：纯 Go library，无 subprocess 依赖；git 依赖拉取无需 buf CLI。

### D3: buf CLI subprocess 拉取 BSR 依赖

`bufbuild/buf` 的核心命令实现在 `internal/`/`private/` 下不可导入，只能 subprocess 调用。仅 BSR 依赖时触发。

**理由**：BSR 协议专有，buf CLI 不可作为 library；subprocess 是唯一方案。无 BSR 依赖时零 buf CLI 依赖。

### D4: 双锁机制

- `api.lock`（git 依赖，apigen 自管）：记录 url/ref/resolved_commit/subdir
- `buf.lock` + `buf.yaml`（BSR 依赖，buf CLI 自管）：v2 格式

**理由**：git 依赖由 apigen 内部 go-git 管理，需独立锁文件；BSR 依赖复用 buf CLI 原生锁机制。

### D5: type_ 视为不透明符号

工具不理解数据模型内容，只做服务层机械拼装。不解析 `resource type_` 内部字段、不解析 `resource.pattern`、不推导资源层级。

**理由**：保持用户数据模型纯净；校验大幅精简；字段号问题天然消失。

### D6: 统一包装原则

所有标准方法的 Request 一律生成 wrapper。Response 视方法/策略而定：Get/BatchGet/List/Create 生成 wrapper；Update 视 version.kind 返回 wrapper（STRONG 回带新版本）或 Empty；Delete/DeleteSoft 返回 Empty。

**理由**：用户资源/主键类型永不直接充当接口参数，数据模型完全纯净。

### D7: 实体级 vs 资源级方法粒度

- Create/Delete/DeleteSoft = 实体级（操作整个实体身份）
- Get/BatchGet/List/Update = 资源级（操作单个数据面）

**理由**：Create 一次创建全部资源；Delete 删除整个实体；读写操作按数据面分离。

### D8: Create 只返回 key（有意取舍）

偏离 AIP-133（建议返回完整资源）。理由：Create 支持部分创建（各资源可选），返回完整资源语义不明确；客户端拿到 key 后按需 Get 具体资源即可。

### D9: WEAK version 用 wrapper 类型

WEAK 版本字段用 `google.protobuf.*Value` wrapper（UInt64Value/UInt32Value/StringValue），可区分"未设置 vs 零值"。STRONG 用标量。

**理由**：proto3 标量无法区分未设置 vs 零值，WEAK CAS 需要这种区分。

### D10: subprocess 安全约定

所有 subprocess 调用使用 `exec.Command` 参数数组形式（非 shell 字符串拼接）；用户输入字段做白名单校验（仅允许 `[a-zA-Z0-9._/:-]`）。

**理由**：杜绝命令注入。

### D11: dep prune 文件级反查

基于文件级依赖反查（非符号级）。protocompile 解析阶段已建立 `.proto 文件 → 来源目录`映射，来源目录对应到具体 `import_protos` 条目。

**理由**：文件级映射在解析时已确定，无符号歧义（同一符号可能被多源提供）。

### D12: filter/order_by 统一 string

List 的 filter/order_by 字段统一用 `string` 类型，不支持自定义 FQMN。filter 语义由服务端解释。

**理由**：简化工具职责；filter 语义因业务而异，工具不预设结构。

## Risks / Trade-offs

### R1: protocompile API 变动风险

protocompile 当前 v0.x，v1 前 API 理论上可能变动。

**对策**：go.mod 锁定版本 + CI 升级测试兜底。核心接口已被 Buf CLI 广泛依赖，breaking 风险可控。

### R2: go-git 对 commit SHA 浅克隆支持有限

go-git library 对 `--depth=1` + 指定 commit SHA 的组合支持有限。

**对策**：ref 为 branch/tag → 浅克隆；ref 为 commit SHA → 完整 clone 后 checkout。

### R3: buf CLI 外部依赖

BSR 依赖需要 buf CLI，引入外部二进制依赖。

**对策**：仅 BSR 依赖时触发；自动检测/安装（`go install`）；版本受 `APIGEN_BUF_VERSION` 控制；无 BSR 依赖时零 buf 依赖。

### R4: Create 只返回 key 偏离 AIP-133

客户端需二次 Get 获取完整资源。

**对策**：有意取舍，部分创建语义下返回完整资源不明确；api-linter 豁免 `core::0133::response-body`。

### R5: subprocess 调用插件的外部依赖

`protoc-gen-go` / `protoc-gen-go-grpc` 需 subprocess 调用，生成逻辑在 internal 包无法导入。

**对策**：优先用 PATH 已有插件；未装则 `go install` 到 GOPATH/bin，版本由 apigen `go.mod` 锁定。

### R6: option 注入纯搬运不校验语义

工具不理解 option 语义，可能搬运非法值。

**对策**：值类型合法性由 protocompile 校验与 extend 声明类型匹配；option 全限定名可达性 + target.path 存在性校验。
