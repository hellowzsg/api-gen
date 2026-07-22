# apigen-optimization Specs

## ADDED Requirements

### Requirement: build 单次执行流（Pipeline 复用）

`apigen build` 全流程中，api.yaml 解析、依赖拉取（git/BSR）、本地 proto glob、protocompile 编译各自最多执行一次。

#### Scenario: build 不重复解析与编译

- **WHEN** 用户对含 git 与 path 依赖的项目执行 `apigen build`
- **THEN** api.yaml 只被解析一次；每个 git 依赖的 `git rev-parse`/clone 至多执行一次；每个 path 目录树只被 glob 一次；同一 proto 文件全流程只被编译一次（user protos 编译一次用于构建 IR；渲染后生成的 service protos 编译一次用于插件，user protos 以全链接描述符复用不重复编译）；最终产物与改造前逐字节一致

#### Scenario: 多个 path import 全部参与编译

- **WHEN** api.yaml 的 `import_protos` 声明两个及以上 `path` 条目，且第二个 path 中的 proto 定义了被 `type_` 引用的消息
- **THEN** 所有 path 条目的 proto 文件都参与 protocompile 编译与类型校验，`apigen generate` 成功且类型引用解析正确

#### Scenario: 第二个 path 中的非法 proto 报错

- **WHEN** 第二个 `path` 条目包含语法错误的 proto 文件
- **THEN** `apigen generate` fail-fast 并报告该文件的编译错误（而非静默忽略）

#### Scenario: api.lock 损坏时报错

- **WHEN** `api.lock` 存在但内容不是合法 YAML
- **THEN** `apigen generate` 返回明确的 lock 解析错误，而非静默回退到 moving-ref 缓存

### Requirement: 依赖解析接口统一与 BSR 批量化

三类依赖（path/git/bsr）通过统一 `dep.Resolver` 接口接入；单次执行中所有 BSR 依赖合并处理。

#### Scenario: 多 BSR 依赖单次 buf dep update

- **WHEN** api.yaml 声明两个及以上 `bsr` 依赖
- **THEN** 全流程只生成一份临时 buf.yaml（含全部依赖），`buf dep update` 只执行一次，逐模块 `buf export`，且用户工作区不出现 `buf.yaml`/`buf.lock` 文件

#### Scenario: BSR export 缓存命中

- **WHEN** 同一 BSR 模块版本已被 export 过（缓存目录存在且 buf.lock digest 一致）
- **THEN** 跳过该模块的 `buf export` 子进程调用，直接使用缓存目录

#### Scenario: BSR version 字段生效

- **WHEN** api.yaml 的 bsr 条目声明 `version: v1.2.3`
- **THEN** `buf export` 以 `module:version` ref 执行，拉取指定版本

### Requirement: 插件声明式注册与并行编译

`build.Compile` 接受 `[]PluginSpec`，插件之间并行执行，不再按配置硬编码 if 块。

#### Scenario: 多插件并行执行且产物完整

- **WHEN** HTTP 与 JS 均启用时执行 `apigen build`
- **THEN** protoc-gen-go、protoc-gen-go-grpc、protoc-gen-grpc-gateway、protoc-gen-openapiv2、protoc-gen-es 全部执行成功，各自产物文件齐全，与串行执行产物一致

#### Scenario: 单插件失败快速返回

- **WHEN** 任一插件执行失败（如未安装）
- **THEN** Compile 返回带插件名与 stderr 的错误，且错误信息可定位到具体插件

#### Scenario: 新增插件不改 Compile 签名

- **WHEN** 未来新增一个 protoc 插件类型
- **THEN** 只需在 cli 组装 `PluginSpec` 处追加一项，`build.Compile` 函数签名不变

### Requirement: HTTP 路径结构化生成

IR 中的 HTTP 注解以结构化段表示，渲染期按当前 service 直接拼接，无字符串重写启发式。

#### Scenario: 同一实体挂到两个 service 路径各自独立

- **WHEN** 实体 `book` 同时被 `LibraryService` 与 `AdminService` 引用且 HTTP 启用
- **THEN** 两个 service 生成的 proto 中 HTTP 路径分别包含各自 service 段，且与改造前输出逐字节一致

#### Scenario: override 路径原样保留

- **WHEN** 资源的 `reader.http.path` 显式声明为 `/custom/{key.id}/items`
- **THEN** 生成 proto 中该路径逐字出现，service 段不被替换

#### Scenario: override 路径非法 key 变量报错

- **WHEN** override path 中包含 `{key.nonexistent}`，该路径不是 key 类型的可达标量叶子
- **THEN** `apigen generate` fail-fast，错误信息指出非法变量与所在资源/方法

#### Scenario: 多段 key 前缀路径拼接正确

- **WHEN** HTTP 启用且 `prefix: /library`，key 类型为复合 key（如 `{org.oid, id}`）
- **THEN** Get 路径为 `/library/<Service>/<entity>/{key.org.oid}/{key.id}/<resource>`，与改造前输出一致

### Requirement: 生成产物可重现

相同 api.yaml 与依赖版本，两次执行生成的 proto 文件内容逐字节相同。

#### Scenario: option map 值输出稳定

- **WHEN** 资源 `options` 中包含 map 类型 value（如 `{"b": 2, "a": 1}`）
- **THEN** 两次 `apigen generate` 输出的 option 文本完全一致（map key 按字典序输出）

### Requirement: 描述符查找索引化

`CompositeResolver` 在编译完成后一次性构建 FQN→descriptor 索引，类型校验与 key descriptor 查询为 O(1)。

#### Scenario: 大量实体类型校验正确性不变

- **WHEN** 项目含 50+ 实体且每个实体 3+ 资源
- **THEN** `validateTypeReferences` 与 key descriptor 查询结果与线性扫描实现完全一致，且单测全部通过

### Requirement: dep prune 真实判定

`apigen dep prune` 基于 type→proto file→依赖归属判定移除未引用的 git 依赖。

#### Scenario: 未引用依赖被移除

- **WHEN** api.lock 含两个 git 依赖，其中一个提供的 proto 未被任何 `key.type_`/`resources[].type_` 引用
- **THEN** prune 后 api.lock 只保留被引用的依赖

#### Scenario: 无法判定时保守保留

- **WHEN** 某 git 依赖的引用归属无法判定（如 type 映射缺失）
- **THEN** 该依赖被保留，不误删

### Requirement: entity list 在 HTTP 启用时可用

`apigen entity list` 为展示命令，不因缺少 key descriptor 而失败。

#### Scenario: HTTP 启用时 entity list 正常输出

- **WHEN** api.yaml 启用 `settings.http.enable: true`，执行 `apigen entity list`
- **THEN** 正常列出实体/资源/方法清单，不报错

### Requirement: 死代码清除

已识别的死代码与重复实现被移除，代码库无无主导出符号。

#### Scenario: lint 包与死方法移除后构建测试通过

- **WHEN** 删除 `internal/lint` 整包、`CompositeResolver` 五个死方法、`ir.buildCreateAnnotation`、`cli.toSnakeCase` 重复实现后
- **THEN** `go build ./...`、`go vet ./...`、全量 `go test ./...` 通过，且无私有引用残留

### Requirement: 关键节点日志

工具在 Pipeline 阶段、依赖拉取、插件编译等关键节点输出结构化日志（slog → stderr），默认静默，`-v` 开启。

#### Scenario: 默认静默

- **WHEN** 用户执行 `apigen generate`（不带 `-v`）
- **THEN** stderr 无 Info 级日志输出，命令行为与现状一致

#### Scenario: verbose 模式输出关键节点

- **WHEN** 用户执行 `apigen build -v`
- **THEN** stderr 出现结构化日志，覆盖：配置解析完成、每个依赖 fetch start/done（含 cache_hit）、protocompile 文件数、IR 构建完成、每个插件 start/done（含 duration）、输出提交

#### Scenario: 日志不污染 stdout

- **WHEN** 用户执行 `apigen entity list -v | grep Entity`
- **THEN** stdout 只含实体清单，日志全部在 stderr，管道结果不受污染

### Requirement: golden 产物兼容回归

examples 作为 golden 基线，验证重构后产物兼容性。

#### Scenario: examples 产物逐字节一致

- **WHEN** 对 `examples/book` 与 `examples/simple` 执行 `apigen build`
- **THEN** 生成产物与改造前 git 基线 `git diff` 为空（预期差异仅限：option map 排序稳定化、JS stub 内嵌 descriptor 字段序稳定化、多 path 场景新增产物——逐项人工核对登记）
