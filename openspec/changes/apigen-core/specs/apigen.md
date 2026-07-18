## ADDED Requirements

### Requirement: 四段式 YAML 解析

apigen 必须解析四段式 `api.yaml`（`import_protos` / `settings` / `entities` / `services`），对 schema 做完整校验，校验失败时 fail-fast 并输出明确错误。

#### Scenario: 合法 api.yaml 解析成功
- **WHEN** 用户提供符合 schema 的 `api.yaml`（含 syntax/name/import_protos/settings/entities/services）
- **THEN** apigen 成功解析为内部配置结构，无错误

#### Scenario: 缺少必填段
- **WHEN** `api.yaml` 缺少 `entities` 段
- **THEN** apigen fail-fast，输出错误提示缺少必填段 `entities`

#### Scenario: name 字段非法
- **WHEN** `api.yaml` 的 `name` 字段不是合法的 proto package 名（如含空格或特殊字符）
- **THEN** apigen fail-fast，输出错误提示 name 字段非法

### Requirement: 实体级方法生成

apigen 必须为每个实体生成实体级方法：`Create<Entity>`、`Delete<Entity>`、`Delete<Entity>Soft`（可选）。

#### Scenario: Create 方法生成
- **WHEN** 实体声明 `create: {}`
- **THEN** 生成 `Create<Entity>` RPC，Request 内嵌各资源字段（按声明序 1..N，均可选），Response 只含 `key=1`

#### Scenario: Delete 方法生成
- **WHEN** 实体声明 `delete: {}`
- **THEN** 生成 `Delete<Entity>` RPC，Request 含 `key=1`，Response 为 `google.protobuf.Empty`

#### Scenario: DeleteSoft 方法生成
- **WHEN** 实体声明 `delete_soft: {}`
- **THEN** 生成 `Delete<Entity>Soft` RPC，Request 含 `key=1`，Response 为 `google.protobuf.Empty`；标记字段由用户 resource proto 定义，工具不生成

#### Scenario: DeleteSoft 与 Delete 共存
- **WHEN** 实体同时声明 `delete: {}` 和 `delete_soft: {}`
- **THEN** 同时生成 `Delete<Entity>` 和 `Delete<Entity>Soft` 两个独立 RPC

### Requirement: 资源级方法生成

apigen 必须为每个资源的 reader/writer 生成资源级方法：Get/BatchGet/List/Update。

#### Scenario: Get 方法生成
- **WHEN** 资源声明 `reader: {}`（默认）
- **THEN** 生成 `Get<Entity><Resource>` RPC，Request 含 `key=1`，Response 含 `<resource>=1` 和 `version=2`（kind≠NONE 时）

#### Scenario: BatchGet 方法生成
- **WHEN** 资源声明 `reader.batch: true`
- **THEN** 生成 `BatchGet<Entity><Resource>s` RPC，Request 含 `repeated <Key> keys=1`，Response 含 `repeated <resource>s=1`

#### Scenario: List 方法生成
- **WHEN** 资源声明 `reader.list: true`
- **THEN** 生成 `List<Entity><Resource>s` RPC，Request 含 `page_size=1, page_token=2, filter=3, order_by=4`（filter/order_by 统一 string），Response 含 `<resource>s=1, next_page_token=2, total_size=3`（total_size 可关闭）

#### Scenario: Update 方法生成
- **WHEN** 资源声明 `writer.update: {}`
- **THEN** 生成 `Update<Entity><Resource>` RPC，Request 含 `<resource>=1, key=2, update_mask=3, version=4`（version 在 kind≠NONE 时），Response 为 `Empty` 或 `{version=1}`（STRONG 时）

### Requirement: 版本/乐观锁策略

apigen 必须根据 `version.kind`（STRONG/WEAK/NONE）和 `version.type`（U64/U32/STRING）生成对应的版本字段。

#### Scenario: STRONG 版本
- **WHEN** 资源声明 `version: { kind: STRONG, type: U64 }`
- **THEN** version 字段为 `uint64` 标量；Get 响应回带 version；Update 请求必带 version 做 CAS；Update 响应回带新 version

#### Scenario: WEAK 版本
- **WHEN** 资源声明 `version: { kind: WEAK, type: U64 }`
- **THEN** version 字段为 `google.protobuf.UInt64Value` wrapper 类型（可区分未设置 vs 零值）；CAS 可选

#### Scenario: NONE 版本
- **WHEN** 资源声明 `version: { kind: NONE }`
- **THEN** wrapper 中无 version 字段

### Requirement: 依赖管理三路径

apigen 必须支持三种依赖类型（path/git/bsr），按类型选择拉取路径。

#### Scenario: path 依赖解析
- **WHEN** `import_protos` 含 `path: "proto/**/*.proto"`
- **THEN** apigen 用 protocompile `SourceResolver` 直接解析，无需 buf CLI

#### Scenario: git 依赖拉取
- **WHEN** `import_protos` 含 `git: <url>, ref: <ref>, subdir: <dir>`
- **THEN** apigen 用 `go-git` library clone 到 `.apigen_cache/git/<hash>/`（branch/tag 浅克隆，commit SHA 完整 clone + checkout），提取 subdir 下 .proto 文件，记录 url+resolved_commit 到 `api.lock`

#### Scenario: bsr 依赖拉取
- **WHEN** `import_protos` 含 `bsr: buf.build/<owner>/<repo>`
- **THEN** apigen 生成 `buf.yaml(v2)`，subprocess 调用 `buf dep update` + `buf export` 到 `.apigen_cache/bsr/<name>/`

#### Scenario: 无 BSR 依赖时零 buf 依赖
- **WHEN** `import_protos` 仅含 path + git 条目
- **THEN** apigen 不调用 buf CLI，不生成 buf.yaml/buf.lock

### Requirement: 双锁机制

apigen 必须维护双锁文件确保可重现构建。

#### Scenario: api.lock 生成（git 依赖）
- **WHEN** 存在 git 依赖且首次拉取
- **THEN** 生成 `api.lock`，记录每个 git 依赖的 url/ref/resolved_commit/subdir

#### Scenario: buf.lock 生成（BSR 依赖）
- **WHEN** 存在 BSR 依赖且首次拉取
- **THEN** buf CLI 生成 `buf.lock`（v2），记录每个 BSR 模块的 commit/digest

#### Scenario: 增量缓存
- **WHEN** `api.lock`（git）或 `buf.lock`（BSR）已存在且与 api.yaml 一致
- **THEN** 跳过拉取，直接用缓存；`apigen dep update` 才强制重新拉取

### Requirement: 通用 option 注入

apigen 必须支持通用 option 注入机制（field/message/rpc/service/file 五档），纯搬运不解析语义。

#### Scenario: field 级 option 注入
- **WHEN** 资源声明 `options: [{ target: field, path: meta, option: acme.cache, value: false }]`
- **THEN** 生成的 wrapper 消息中对应字段添加 `[(acme.cache) = false]`

#### Scenario: option 全限定名可达性校验
- **WHEN** 声明的 option 全限定名在 import 闭包中找不到对应 `extend` 定义
- **THEN** apigen fail-fast（protocompile link 阶段校验）

#### Scenario: target.path 存在性校验
- **WHEN** `target: field, path: meta` 但 wrapper 内不存在该字段
- **THEN** apigen fail-fast

### Requirement: proto 生成与确定性输出

apigen 必须生成符合 AIP 风格的服务层 proto，输出 bit-identical。

#### Scenario: proto 生成
- **WHEN** `apigen generate` 成功
- **THEN** 在 `generated/proto/<service>/<service>.proto` 生成 proto 文件，含 service 定义、RPC 方法、wrapper 消息、HTTP 注解（P0 不生成）

#### Scenario: 确定性输出
- **WHEN** 同一 `api.yaml` 多次运行 `apigen generate`
- **THEN** 生成的 proto 文件 bit-identical（import 排序字典序、message/RPC 顺序固定、字段号固定）

### Requirement: 编译成 stub

apigen 必须能将生成的 proto 编译成 Go stub（`*.pb.go` / `*_grpc.pb.go`）。

#### Scenario: build 成功
- **WHEN** `apigen generate` 成功后运行 `apigen build`
- **THEN** subprocess 调用 `protoc-gen-go` 和 `protoc-gen-go-grpc`，在 `generated/go/<service>/` 下生成 `*.pb.go` 和 `*_grpc.pb.go`

#### Scenario: 可编译性形式化保证
- **WHEN** `apigen generate` 成功
- **THEN** `apigen build` 的 proto→stub 编译必然成功（不会因 import/依赖失败）

### Requirement: 后置校验

apigen 必须支持 api-linter 和 buf breaking 后置校验。

#### Scenario: api-linter 豁免生成
- **WHEN** 生成的 proto 偏离 AIP 条款
- **THEN** 在 proto 顶部（syntax 声明之前）按实际触发裁剪插入文件级行内豁免注释

#### Scenario: buf breaking（仅 BSR 依赖时）
- **WHEN** 存在 BSR 依赖且 `buf.yaml` 配置了 breaking 规则
- **THEN** apigen 调用 `buf breaking` 做兼容性校验

### Requirement: CLI 子命令

apigen 必须提供 generate/build/dep/entity 子命令。

#### Scenario: generate 命令
- **WHEN** 运行 `apigen generate -f api.yaml`
- **THEN** 执行 schema 校验 → 依赖拉取 → protocompile 解析 → 语义校验 → IR 构建 → 模板渲染，仅生成 proto

#### Scenario: build 命令
- **WHEN** 运行 `apigen build -f api.yaml`
- **THEN** 执行 generate 全流程 + 编译成 pb.go/grpc.pb.go

#### Scenario: dep update 命令
- **WHEN** 运行 `apigen dep update`
- **THEN** 强制重新拉取所有远程依赖，刷新 api.lock（git）和 buf.lock（BSR）

#### Scenario: dep prune 命令
- **WHEN** 运行 `apigen dep prune`
- **THEN** 基于文件级依赖反查，移除 api.lock/buf.yaml 中未被引用的远程依赖

### Requirement: type_ 类型约束

apigen 必须校验 `type_` 字段解析为 proto message 类型。

#### Scenario: type_ 为 message 类型
- **WHEN** `key.type_` 或 `resources[].type_` 解析为 proto message 类型
- **THEN** 校验通过

#### Scenario: type_ 为 enum 或标量别名
- **WHEN** `type_` 解析为 enum 或标量别名
- **THEN** apigen fail-fast，提示 type_ 必须为 message 类型

### Requirement: subprocess 安全约定

apigen 的所有 subprocess 调用必须使用参数数组形式，杜绝命令注入。

#### Scenario: 参数数组形式
- **WHEN** apigen 调用 buf CLI 或 protoc-gen-* 插件
- **THEN** 使用 `exec.Command` 参数数组形式（非 shell 字符串拼接）

#### Scenario: 用户输入白名单校验
- **WHEN** 用户输入字段（git/bsr/ref/alias/path）传入 subprocess
- **THEN** 做白名单校验（仅允许 `[a-zA-Z0-9._/:-]`），非法字符 fail-fast
