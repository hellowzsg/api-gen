# apigen-optimization：性能、冗余与可扩展性全面优化

## Why

对当前代码库（`cmd/` + `internal/{cli,build,dep,ir,lint,render,yaml}`）的全量分析发现三类系统性问题：

### 性能瓶颈

1. **`apigen build` 把整个 generate 流程执行两遍**（`internal/cli/build.go:14-44`）：`runGenerate` 完成后，`runBuild` 再次解析 YAML、再次拉取全部依赖（git 子进程 / `buf export`）、再次 glob 本地 proto、再次 protocompile 全量编译。一次 build = 2× YAML 解析 + 2× 依赖拉取 + 4× 本地 glob + 2× proto 编译。
2. **同一 path import 单次 generate 内被 glob 两次**（`generate.go:140` 与 `generate.go:168-175` 各建一个 PathResolver）。
3. **5 个 protoc 插件串行执行，且每次 `proto.Clone` 深拷贝整个 CodeGeneratorRequest**（`compiler.go:257-314`），大项目下内存放大 5 倍。
4. **BSR 依赖逐条处理**：每个 BSR import 单独建 resolver，互相覆盖写入用户工作区的 `buf.yaml`，各自跑 `buf dep update` + `buf export`，且 export 无任何缓存（`generate.go:156-163`、`bsr.go`）。
5. **`CompositeResolver.findDescriptor` 线性扫描**（`composite.go:79-86`）：类型校验与 IR 构建阶段对每个 entity/resource O(files) 查找。

### 正确性缺陷（随性能问题一并修复）

- **多个 `import_protos.path` 时只有第一个被编译**（`setupPathResolver` 对首个命中即 return）：后续 path 的 proto 文件静默缺失于编译与类型校验。
- **`FormatOptionValue` 遍历 map 顺序随机**（`ir/option.go:86-91`）：同一 api.yaml 两次生成内容可能不同，破坏可重现构建。
- **`apigen entity list` 在 HTTP 启用时必然失败**：`ir.Build` 走 `BuildWithOptions` 空 `KeyDescriptors`，`buildEntity` fail-fast。
- **`dep prune` 是 no-op 桩**：`isDepReferenced` 恒返回 true（`cli/dep.go:106-110`）。
- **`runBuild` 多处吞错误**（`_ = pathResolver.Glob()` 等，build.go:30,38,40），与 generate 的 fail-fast 不一致；`api.lock` 解析错误被静默忽略（generate.go:130）。

### 冗余与可扩展性

- `internal/lint` 整包无生产调用方，且豁免生成逻辑与 `render.generateExemptions` 重复。
- `CompositeResolver` 有 5 个无生产调用方的导出方法；`buildCreateAnnotation` 死代码；`toSnakeCase` 两处重复实现；`BSRDep.Version` 死字段。
- `build.Compile` 签名 9 个参数、插件硬编码 if 块，新增插件必须改签名。
- HTTP 注解路径在 IR 构建期烧入"第一个引用 service 名"，渲染期靠 `replaceServiceSegment` 字符串启发式重写（`render/template.go:133-185`），对 override 路径与多段 prefix 脆弱。
- 三类依赖 resolver 接口不统一，新增依赖类型需改 switch。

## What Changes

按方案 A（全面重构）实施五大变更块：

### 1. Pipeline 复用重构（消除双重执行）

- 新增 `internal/cli/pipeline.go`：`Pipeline` 结构封装 `Config → 依赖解析 → protocompile → IR` 的共享产物（cfg、importPaths、pathResolvers、linker.Files、IR）。
- `runGenerate` 改为基于 `Pipeline` 的单次执行；`runBuild` 直接复用同一 `Pipeline` 结果，仅追加种子文件收集与插件编译。
- 修复多 path import：`setupPathResolver` 改为收集**所有** path import 的 resolver。
- `runBuild` 错误处理与 generate 对齐 fail-fast；`api.lock` 仅在文件不存在时忽略，解析错误显式返回。

### 2. 依赖解析统一

- 定义统一 `dep.Resolver` 接口（`Fetch/ImportPaths/ProtoFiles`），path/git/bsr 三实现归一；`resolveDependencies` 改为遍历接口切片。
- BSR 批量化：单次 generate 中所有 BSR 依赖合并为一个 resolver，一次 `buf dep update`；`buf export` 按模块增加内容缓存（export 目录已存在且 buf.lock 未变则跳过）；`buf.yaml`/`buf.lock` 写入临时工作目录，不再污染用户工作区；删除死字段 `BSRDep.Version`（或接入为 export ref，二选一在实施中定）。
- `dep prune` 真实实现：基于 `BuildTypeImportPaths` 的 type→proto file 映射 + file 路径是否位于某 git dep 克隆目录内，判定依赖是否被引用。

### 3. 插件注册表与并行编译

- `build.Compile` 重构为声明式 `PluginSpec{Name, OutDir, Parameter, Enabled}` 列表，由调用方（cli）按配置组装，Compile 只负责遍历执行。
- 插件间用 `golang.org/x/sync/errgroup` 并行执行（`x/sync` 已在 indirect 依赖，提升为直接依赖）。
- 消除深拷贝：各插件请求浅拷贝 `CodeGeneratorRequest` 结构体并替换 `Parameter` 指针，`ProtoFile` 切片只读共享。

### 4. HTTP 路径结构化（消除启发式重写）

- IR 的 `HTTPAnnotation` 改为结构化表示：保留 `Verb/Body/IsOverride`，路径不再存字符串，改存结构化段（entity、keyLeaves、resource、suffix、overridePath）。
- `ir.Build` 不再依赖 `firstServiceForEntity` 烧入 service 名；`render` 按当前渲染的 service 直接拼接完整路径，删除 `replaceServiceSegment` 启发式与 `firstServiceForEntity`。
- `ir.ValidatePathVariables` 接入生产路径：逐方法 http override 的 `{key.*}` 变量在 IR 构建期对照 KeyLeaves 校验，非法变量 fail-fast（补上 P2 遗留 TODO，`yaml/validate.go:177`）。

### 5. 正确性修复与死代码清除

- `FormatOptionValue` 对 map 类型按 key 排序后输出，保证生成内容可重现。
- `CompositeResolver` 在 `Resolve` 后一次性构建 FQN→descriptor 索引，`findDescriptor`/`CheckTypeIsMessage`/`FindMessageDescriptor` 改 O(1)。
- 删除 `internal/lint` 整包（豁免生成由 render 负责；`RunApiLinter`/`RunBufBreaking` 无调用方；未来如需 `apigen lint` 命令单独提案）。
- 删除 `CompositeResolver` 死方法（`DryRunClosure`/`CollectTransitiveClosure`/`ResolveWithFiles`/`AddImportPaths`/`CheckSymbolReachable`）及对应测试；删除 `buildCreateAnnotation` 死代码。
- `toSnakeCase` 归一为 `ir.ToSnakeCase` 导出，cli 复用。
- `entity list` 修复：`BuildWithOptions` 增加 lenient 模式（HTTP 启用但无 KeyDescriptors 时跳过 key-leaf 提取），展示类命令不再失败。

### 6. 关键节点日志

当前工具全程静默（除最终错误外无任何进度输出），依赖拉取与插件编译耗时稍长时用户无法感知进展，排查问题只能逐层加打印。本次引入结构化日志：

- 标准库 `log/slog`（Go 1.24 内置，零新增依赖），输出到 **stderr**（stdout 保留给 `entity list` 等命令结果）。
- 默认静默（Warn 级，保持现有 CLI 体验）；新增全局 `-v/--verbose` flag（或 `APIGEN_LOG_LEVEL`）开启 Info/Debug。
- 关键节点：Pipeline 各阶段（配置解析、每个依赖 fetch start/done/缓存命中、protocompile 文件数、IR 实体数、渲染/提交）、每个插件 start/done（含 duration）与失败详情、BSR/git 缓存命中跳过。

### 兼容性与验证基线

- CLI 命令、`api.yaml` schema、生成产物目录布局**全部不变**。
- 除以下预期行为变化外，相同输入的生成产物必须与现状**逐字节一致**（以 `examples/book`、`examples/simple` 为 golden 回归）：
  1. 多 path import 的 proto 现在真正参与编译（可能暴露此前被静默跳过的 proto 错误 → 正确的 fail-fast）；
  2. 生成输出稳定化：option map 值按 key 排序；plugin 请求确定性 marshal 修复后 JS stub 内嵌 descriptor 的 `post`/`body` 字段序固定（旧流程随机，HEAD golden 即不自洽）；
  3. BSR 流程不再在用户工作区生成/覆盖 `buf.yaml`、`buf.lock`。
- 附带修复的 pre-existing 缺陷（实施中发现，均属正确的 fail-fast 或确定性改进）：仅由 git/BSR 依赖提供的实体类型现在可正常解析（旧代码传递 import 文件不参与描述符查找）；`api.lock` 损坏从静默回退改为显式报错。

## Impact

### 修改代码

- `internal/cli/pipeline.go`（新增）：Pipeline 结构与共享执行逻辑（Prepare + resolveDependencies 接口遍历 + BSR 聚合）
- `internal/cli/generate.go`：runGenerate 基于 Pipeline；toSnakeCase 去重（复用 ir.ToSnakeCase）
- `internal/cli/build.go`：runBuild 复用 Pipeline + ResolveExtra（生成 proto 复用已编译描述符）；pluginSpecsForConfig 组装
- `internal/cli/dep.go`：dep prune 真实实现（Prepare + type→file→cloneDir 归属判定）；entity list lenient
- `internal/cli/logging.go`（新增）：slog 初始化（-v / APIGEN_LOG_LEVEL）
- `internal/cli/root.go`：全局 `-v/--verbose` flag
- `internal/dep/resolver.go`（新增）：统一 Resolver 接口（Fetch(ctx) + ProtoFiles）
- `internal/dep/bsr.go`（重写）：批量 Fetch + buf.lock digest export 缓存 + 临时工作目录 + version ref
- `internal/dep/git.go`：Fetch(ctx) 全链 CommandContext；CloneDir；缓存命中日志
- `internal/dep/composite.go`：FQN 索引；传递闭包查找修复；ResolveExtra；删除 5 个死方法
- `internal/build/compiler.go`：PluginSpec 注册表 + errgroup 并行 + 浅拷贝 + 确定性 marshal
- `internal/ir/builder.go`：HTTPAnnotation 结构化（ResolvePath）；ValidatePathVariables 接入；LenientHTTP；删除 firstServiceForEntity/buildCreateAnnotation
- `internal/ir/option.go`：FormatOptionValue map key 排序
- `internal/render/template.go`、`internal/render/http.go`：按结构化路径段拼接；删除 replaceServiceSegment/rewriteHTTPPathForService
- `internal/yaml/validate.go`：P2 遗留 TODO 清除
- `internal/lint/`（整包删除，4 文件）
- 测试：pipeline/resolver/bsr/dep_prune/logging 新增；composite/ir/render/build 测试同步改造
- `go.mod`：`golang.org/x/sync` 提升为直接依赖

### 测试

- 上述各包 `_test.go` 同步更新；新增 pipeline、resolver 接口、BSR 缓存、并行编译、HTTP 路径拼接、option 排序的测试
- golden 回归：`examples/book`、`examples/simple` 全量 build 后 `git diff` 必须为空（除三项预期行为变化）

### 依赖

- `golang.org/x/sync` 从 indirect 提升为直接依赖（errgroup）

### 用户侧影响

- `apigen build` 耗时预期下降约 40-50%（消除一次完整 generate + 插件并行 + BSR 缓存）
- 使用多个 `import_protos.path` 的项目：此前静默缺失的 proto 现在参与编译，可能暴露新错误（正确的 fail-fast）
- 用户工作区不再出现 apigen 生成的 `buf.yaml`/`buf.lock`
