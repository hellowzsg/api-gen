# apigen-optimization 设计文档

## Context

apigen 已完成 P0（纯 gRPC）→ P1（HTTP MVP）→ P2（HTTP 增强 + JS stub）三轮演进，功能面快速扩张的同时积累了性能与结构债务。本设计基于对 `cmd/` + `internal/` 全部 45 个 Go 文件的逐文件审查（审查结论见 proposal.md Why 节），在不改变 CLI 界面、`api.yaml` schema、生成产物布局的前提下，对内部实现做全面优化。

关键现状：

- `runBuild` 与 `runGenerate` 是两个独立实现的全流程，build = generate + 重复解析 + 编译（`internal/cli/build.go:14-44`）。
- 三类依赖 resolver（path/git/bsr）各自为政，BSR 每依赖一次 `buf dep update` 且写用户工作区 `buf.yaml`。
- `build.Compile` 9 参数硬编码 5 个插件，串行 + 每次全量 `proto.Clone`。
- HTTP 路径在 IR 构建期用 `firstServiceForEntity` 烧入 service 名，渲染期 `replaceServiceSegment` 启发式重写（`render/template.go:133-185`），注释自承认是 heuristic。
- `internal/lint` 整包、`CompositeResolver` 5 方法、`buildCreateAnnotation`、`BSRDep.Version` 均为死代码；`lint.GenerateExemptions` 与 `render.generateExemptions` 逻辑重复。

## Goals / Non-Goals

**Goals:**

1. `apigen build` 单次执行流：解析、依赖拉取、proto 编译在整个 build 中只发生一次。
2. 依赖解析接口统一；BSR 批量处理 + export 缓存 + 不污染用户工作区。
3. 插件声明式注册 + 并行执行 + 零深拷贝。
4. HTTP 路径 IR 结构化，删除字符串启发式重写；override path 的 `{key.*}` 变量构建期校验。
5. 生成产物可重现（option map 输出排序）；描述符查找 O(1)。
6. 清除全部已识别死代码；`dep prune` 真实实现；`entity list` HTTP 下可用。
7. 生成产物逐字节兼容（除 proposal.md 列明的三项预期变化），以 examples 为 golden 回归。

**Non-Goals:**

- 不改变 CLI 命令集、参数、`api.yaml` 任何字段语义（纯内部重构）。
- 不引入新功能（如 `apigen lint` 命令、OpenAPI/JS 之外的插件类型）。
- 不重构 examples/ 业务 proto 与 README 内容（仅作为 golden 基线使用）。
- 不做 HTTP 功能语义演进（AIP-136 custom method 路由、LRO 等属后续提案）。
- 不改 api.lock 文件格式。

## Decisions

### D1：Pipeline 单次执行流

新增 `internal/cli/pipeline.go`：

```go
type Pipeline struct {
    Config       *apigenyaml.Config
    BaseDir      string
    ImportPaths  []string
    PathResolvers []*dep.PathResolver   // 所有 path import（修复多 path bug）
    Resolver     *dep.CompositeResolver  // 已 Resolve
    IR           *ir.IR                  // 含 TypeImportPaths
}

func Prepare(ctx context.Context, apiYAMLPath string) (*Pipeline, error)
```

- `runGenerate` = `Prepare` + 渲染 staging + 原子提交。
- `runBuild` = `Prepare` + 渲染提交 + `Compile`（复用 `Pipeline.Resolver.Files()`，不再第二次 parse/resolve）。
- 备选方案（build 内嵌 generate 结果缓存到结构体而非函数返回）被否决：显式返回值更清晰，测试可直接构造 Pipeline 局部状态。
- `api.lock` 读取错误处理：仅 `os.IsNotExist` 忽略，其他错误（YAML 解析失败）显式返回——静默忽略会掩盖 lock 损坏并回退到 moving-ref 缓存。

**实现补充（任务组 1 落地细节）：**

- `runBuild` 的"复用"通过 `CompositeResolver.ResolveExtra(importPaths, files)` 实现：渲染后的生成 proto 单独编译，user protos 以 `protocompile.SearchResult{Desc}` 全链接描述符形式复用——每个 proto 文件全流程只编译一次（spec 场景"protocompile 只执行一次"按文件粒度兑现；compiler 调用为两次，因生成 proto 在渲染前不存在，无法与 user protos 同批编译）。
- **发现并修复 pre-existing 可重现性缺陷**：`RunPlugin` 的 `proto.Marshal(req)` 对含 dynamicpb 扩展（google.api.http option）的请求字段序随机，导致同一 req 分发给各插件的字节流互不一致（HEAD golden 即不自洽：`.pb.go` 为 [body,post]、`.ts` 为 [post,body]）。修复为 `proto.MarshalOptions{Deterministic: true}`，6 次运行字节级稳定。`.ts` 与 HEAD 的字段序差异登记为预期稳定化差异（与 proposal 申报的第 2 项同类）。
- 测试 seam：`cli.prepareFn`/`cli.compileFn` 包级变量，验证 runBuild 单次 Prepare 且无需真实插件。

### D2：统一 dep.Resolver 接口

```go
type Resolver interface {
    Fetch(ctx context.Context) (importPaths []string, err error)
    ProtoFiles() []string   // 本地文件列表（path/git 有；bsr 无，返回 nil）
}
```

- `resolveDependencies` 从 switch 改为：构造 `[]Resolver`（保持配置声明顺序），顺序执行（网络 I/O，可后续再议并行；本次保持顺序避免 git/buf 并发写缓存的竞态）。
- `CompositeResolver` 持有全部 `ProtoFiles()` 非空的 resolver 文件列表 + 聚合 importPaths，一次 protocompile。
- 接口统一后新增依赖类型（如 OCI）只需新增实现，不改 cli。

**实现补充（任务 2.1 落地细节）：**

- `ProtoFiles()` 语义修正为"**必须显式命名编译的本地文件**"：仅 path 返回 glob 结果；git/BSR 返回 nil——其 proto 仅作为传递 import 惰性编译，显式命名会把整个依赖仓库（如 googleapis 数千文件）纳入每次编译（行为变更 + 性能灾难）。设计稿"path/git 有"按此修正。
- `Fetch(ctx)` 全链接入 `exec.CommandContext`（git clone/checkout/rev-parse、buf dep update/export），支持取消。
- `GitResolver.ProtoFiles() ([]string, error)` 旧方法删除（无生产调用方）；git 测试改用同包字段断言。
- `AddPathResolver` 增加 import path 去重：路径已由 `Fetch` 按声明顺序聚合，避免重复注册。
- 错误文案保持现状：`namedResolver{kind, name}` 在 cli 侧保留 "path/git/bsr dependency %q" 包装。

### D3：BSR 批量化与缓存

- 单次执行收集全部 BSR 依赖为一个 `BSRResolver{deps: []BSRDep}`：
  - `buf.yaml`/`buf.lock` 写入 `os.MkdirTemp` 临时目录（含全部 deps），`buf dep update` 仅一次；不再触碰用户工作区。
  - export 缓存：目标目录 `<cache>/v1/module-proxy/bsr/<module_underscored>/<buf.lock 中该模块 digest>/`；目录已存在则跳过 `buf export`。digest 来自 `buf.lock` 解析，内容寻址保证正确失效。
  - `ImportPaths()` 只返回本次声明模块的 export 目录（修掉当前枚举整个 bsr 缓存目录导致无关模块泄漏为 import path 的问题）。
- `BSRDep.Version`：接入为 `buf export <module>:<version>` 的 ref（原字段死代码问题一并解决）；`api.yaml` 的 `version` 字段语义不变。

### D4：PluginSpec 注册表与并行编译

```go
type PluginSpec struct {
    Name      string  // protoc-gen-go / protoc-gen-go-grpc / ...
    OutDir    string
    Parameter string
}

func Compile(ctx context.Context, files linker.Files, fileToGenerate []string, specs []PluginSpec) error
```

- cli 按配置组装 specs（go 必选；grpc 必选；gateway/openapi 看 HTTP；es 看 plugins.js），`Compile` 不再感知配置语义——新增插件只改 cli 组装处。
- 并行：`errgroup.WithContext(ctx)`，每个 spec 一个 goroutine；输出目录互不重叠（go/grpc/gateway 同目录但文件名后缀不同 `.pb.go`/`_grpc.pb.go`/`.pb.gw.go`，由不同进程写不同文件，无写冲突；openapi/js 各自独立目录）。
- 浅拷贝：每个 goroutine `reqCopy := *req; reqCopy.Parameter = &param`；`ProtoFile` 切片在 marshal 前只读，无数据竞争（`RunPlugin` 内部即 marshal）。预期大项目内存占用从 5×request 降为 1×request。

**实现补充（任务组 3 落地细节）：**

- 结构体浅拷贝 `reqCopy := *req` 会触发 copylocks（`CodeGeneratorRequest` 内嵌 `MessageState` 锁标记，`go vet` 检出）。改为**按字段组装新 request**（`FileToGenerate`/`ProtoFile` 切片只读共享，`Parameter` 各自赋值）——零深拷贝且 vet 干净。
- 输出目录创建下沉到各插件 goroutine（`os.MkdirAll` 并发安全）；错误信息保留插件名定位（`run <name>: ...`）。
- `go.mod`：`golang.org/x/sync` 提升为直接依赖（errgroup）。

### D5：HTTP 路径结构化

- `ir.HTTPAnnotation` 改为：

```go
type HTTPAnnotation struct {
    Verb        string
    Body        string
    IsOverride  bool
    OverridePath string     // IsOverride 时的完整路径（不做 service 段重写）
    Entity      string     // 默认路径用
    KeyLeaves   []KeyLeaf  // 默认路径用
    Resource    string     // 默认路径用（"" 表示实体级方法）
    Suffix      string     // "batchGet" / "list" / "deleteSoft" / ""
}
```

- `ir.Build` 完全不再感知 service；`render.RenderServiceProto` 用 `prefix + svc.Name + 结构段` 直接拼接默认路径；override 路径原样使用（当前对 override 也做重写的启发式逻辑删除——override 语义即"用户显式指定完整路径"）。
- 删除 `firstServiceForEntity`、`replaceServiceSegment`。
- 生成产物兼容性：默认路径拼接规则与现行 P1/P2 输出完全一致（golden 测试保障）；override 路径不再被重写属于行为修复（当前启发式对 override 路径的重写本就是缺陷——用户显式路径不应被篡改）。
- `ValidatePathVariables` 接入：`httpBuildContext.applyOverride` 设置 `OverridePath` 时对照 `keyLeaves` 校验 `{key.*}`，非法立即 fail-fast（补 P2 遗留 TODO）。

**实现补充（任务组 4 落地细节）：**

- `HTTPAnnotation.ResolvePath(prefix, svcName)` 为纯函数：默认路径 `prefix + svc + entity + {key...} + resource + suffix`；override 路径区分两种语义——
  - **verbatim override**（custom method）：`IsOverride=true` 且 `OverrideTemplateSvc=""`，路径原样返回。custom method 只挂在声明它的 service 上，不存在跨 service 继承，无需重写。
  - **template override**（entity 级 `reader.http`/`writer.update.http`）：`IsOverride=true` 且 `OverrideTemplateSvc=<firstServiceForEntity>`。该 entity 可能被多个 service 通过 narrowing 继承，override path 中等于 `OverrideTemplateSvc` 的段在 ResolvePath 时替换为当前渲染 service 名，保证各 service 路由隔离。`firstServiceForEntity` 取 cfg.Services 中第一个引用该 entity 的 service 名作为模板 service（即 api.yaml 中用户书写 override path 时使用的 service 名）。
- `rewriteSegment(path, old, new)` 按 "/" 切段精确字符串匹配首个等于 old 的段替换——比旧 `replaceServiceSegment` 的位置启发式（按 `{` 位置推断 prefix 长度）更稳健，不会误改 entity/resource 段。
- 渲染侧经 `httpRenderContext{prefix, svcName}` 线程化到 `renderRPCWithHTTP`，删除全部字符串重写启发式后渲染不再原地改写共享注解（旧代码在共享 EntityIR 上改 Path，新方法天然并发安全）。
- `applyOverride` 改为返回 error（校验接入），`fillResourceAnnotations`/`buildEntity` 依次传播；签名中未使用的 `hasBody` 参数移除，新增 `methodName` 与 `templateSvc` 参数。
- custom method 注解保持 `IsOverride=true + OverridePath`（AIP-136 冒号语法原样保留，`OverrideTemplateSvc` 留空）。
- **golden 差异登记（修正）**：先前版本误判 override path 应"verbatim 不重写"，导致 `admin_service.proto` 的 ListBookMetas 路径变为 `/library/LibraryService/book/meta/list`（与 LibraryService 路由冲突，e2e_http_test 触发 nil panic + 404）。修正为 template 语义后，`examples/book` 的 `admin_service` ListBookMetas 路径恢复为 `/library/AdminService/book/meta/list`——与 git HEAD 完全一致（无 golden 差异）。

### D6：可重现与索引化

- `FormatOptionValue`：`map[string]interface{}` 分支先 `sort.Strings(keys)` 再拼接。递归嵌套 map 同样有序。
- `CompositeResolver.Resolve` 末尾构建 `descByFQN map[string]protoreflect.Descriptor`（遍历所有文件的消息/枚举/服务/字段递归注册一次）；`findDescriptor` 改 map 查找。`BuildTypeImportPaths` 复用同一索引。

**实现补充（任务组 5 落地细节）：**

- FQN 索引覆盖 messages/enums/services/methods/fields/extensions 递归注册；`BuildTypeImportPaths` 改为遍历索引并过滤非 message/enum 与标准 import（输出与原实现一致，golden 验证）。
- `BuildOptions.LenientHTTP`：`entity list` 等展示命令跳过 key-leaf 提取；generate/build 保持严格。
- **可重现性另修复（任务组 1 期间发现）**：`RunPlugin` 的 `proto.Marshal` 改为 `MarshalOptions{Deterministic: true}`——旧流程对含 dynamicpb 扩展的请求字段序随机，同次 build 分发给不同插件的字节流互不一致（详见 D1 补充）。

### D7：死代码清除与 lint 包处置

- **删除 `internal/lint` 整包**（linter.go/breaking.go + 2 测试文件）：豁免生成唯一权威在 `render.generateExemptions`（生产路径使用）；`RunApiLinter`/`RunBufBreaking` 自 apigen-core 起从未接线。未来若做 `apigen lint` 命令，按新需求单独提案设计（不背历史包袱）。
- 删除 `CompositeResolver` 的 `DryRunClosure`/`CollectTransitiveClosure`/`ResolveWithFiles`/`AddImportPaths`/`CheckSymbolReachable`（均无生产调用方；闭包收集由 `build.BuildCodeGeneratorRequest` 负责）及对应测试用例。
- 删除 `ir.buildCreateAnnotation`（被 `buildCreateAnnotationWithResources` 取代）。
- `toSnakeCase`：`ir` 包导出 `ToSnakeCase`，`internal/cli` 删除本地实现。
- `entity list`：`BuildWithOptions` 增加 `BuildOptions.LenientHTTP bool`——HTTP 启用但缺 KeyDescriptors 时跳过 key-leaf 提取（展示命令不需要 HTTP 路径）；generate/build 路径保持严格。

### D8：dep prune 真实实现

- `Prepare` 流程产生 `BuildTypeImportPaths()`（type→proto file path）与每个 git dep 的克隆目录。
- 判定：被引用 type 的 proto file 绝对路径若前缀匹配某 git dep 的 `<cloneDir>/<subdir>`，则该 dep 被引用。
- prune 时保留被引用 dep + 所有无法判定的 dep（保守策略，宁可保留不误删）；BSR 依赖不在 api.lock 中（lock 只含 git_deps），不涉及。

**实现补充（任务 2.2/2.3 落地细节）：**

- BSR：`NewBSRResolver(deps, cacheDir)` 去掉 workDir 参数（buf.yaml/buf.lock 全部写入 `os.MkdirTemp` 临时目录并 defer 清理）；`buf export` 先到同父临时目录再 rename 原子发布；缓存 key = buf.lock digest（`:` 替换为 `_`）；`BSRDep.Version` 接入 `module:version` ref 并加白名单校验（`^[a-zA-Z0-9._-]+$`）；`ImportPaths()` 只返回本次声明模块（修复全缓存目录枚举泄漏）。`GenerateBufYAML`/`HasDeps` 公共方法随旧实现删除。
- **发现并修复 pre-existing 类型解析缺陷**：`protocompile.Compile` 只返回显式命名的文件，`findDescriptor`/`BuildTypeImportPaths` 此前仅遍历该集合，导致**仅由 git/BSR 依赖提供的实体类型无法解析**（generate 即失败）。修复：新增 `allFiles()`（命名文件 + 传递 import 闭包 BFS），查找与类型映射均遍历闭包；`BuildTypeImportPaths` 排除标准 import（WKT/annotations）保持 map 内容与原行为一致（golden 验证无差异）。
- prune 判定中"无法判定"分支：clone 目录不在缓存（未拉取或已清理）或 type 不在编译映射中 → 保守保留。`GitResolver.CloneDir()` 新增（纯路径计算，无网络）。

### D9：关键节点日志（slog）

- 选型：标准库 `log/slog`（Go 1.24 内置，零新增依赖）；不引入第三方日志库。
- 输出：**stderr**。stdout 保留给 `entity list` 等命令结果，管道场景不被日志污染。
- 级别控制：root command 增加持久化 `-v/--verbose` flag（Info）；`APIGEN_LOG_LEVEL=debug` 可进一步打开 Debug；默认 Warn（与现状一致的静默体验）。`PersistentPreRun` 中按 flag/env 构造 `slog.NewTextHandler` 并 `slog.SetDefault`。
- 插桩点（Info 级，均带 `stage`/`duration` 等结构化属性）：
  - Pipeline：`config parsed`（entities/services 数）、每个依赖 `dep fetch start/done`（kind、url/module、`cache_hit`）、`protos resolved`（文件数、耗时）、`IR built`（实体/服务数）、`proto rendered`（服务数）、`output committed`
  - 插件：`plugin start/done`（name、outDir、duration）；失败时 Error 级含 stderr 摘要
  - 缓存：git clone 命中、BSR export 命中（Info，`cache_hit=true`）
- 测试：通过 `slog.SetDefault` 替换为 buffer handler 断言关键节点日志出现；默认级别下无输出。

**实现补充（任务组 6 落地细节）：**

- `entity list`/`dep update`/`dep prune` 不经完整 Prepare 流水，无专属插桩（spec 场景以 `build -v` 为准）；如需 dep 命令日志可后续补充。
- 属性 key 约定：`kind`/`name`/`files`/`entities`/`services`/`plugin`/`outDir`/`duration`/`cache_hit`/`module`/`url`；缓存命中统一 `dep fetch cache hit` + `kind`（git/bsr）。

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| 多 path 修复后，旧配置中"第二个 path 里编译不过的 proto"开始报错 | 这是正确的 fail-fast；proposal Impact 节显式声明；golden examples 覆盖单 path + 多 path 场景 |
| 插件并行后错误输出交错 | errgroup 收集首个错误；各插件 stderr 在各自 error 中包装，不共享终端 |
| 浅拷贝共享 `ProtoFile` 有数据竞争 | request 在 `RunPlugin` 入口即 marshal，goroutine 内无写操作；竞态检测 `go test -race` 纳入验证 |
| HTTP 路径结构化改动面大，产物回归 | examples/book（HTTP enable + prefix + override）与 examples/simple（纯 gRPC）双 golden：`go run ./cmd/apigen build -f ...` 后 `git diff --exit-code`；override 路径不再重写属预期差异，需逐条人工核对 |
| BSR 缓存 key 设计错误导致 stale export | digest 取自 `buf.lock`（buf 自己维护的内容寻址），apigen 不自行发明失效逻辑 |
| errgroup 提升 `x/sync` 为直接依赖 | 已在 go.sum（indirect v0.8.0），无新增外部依赖 |
| dep prune 误删 | 保守策略：无法判定归属的 dep 一律保留；prune 输出删除清单供用户确认（保持现有命令行为，仅把恒 true 桩换成真实判定） |
