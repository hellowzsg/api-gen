## Why

微服务基于 gRPC 构建，需统一 proto 接口定义规范。手写服务层 proto 存在大量样板代码（Request/Response 包装、分页、update_mask、版本 CAS、依赖拉取、编译），且易出错（字段号冲突、import 闭包不完整、依赖漂移）。

现有工具（buf、protoc）解决编译/兼容性问题，但不解决**服务层样板生成**问题。用户需要一种声明式方式描述"业务对象有哪些资源、每个资源的读写策略"，由工具自动生成符合 Google AIP 资源导向设计的服务层 proto 并一键编译成 stub。

## What Changes

构建 **apigen** CLI 工具（纯 Go），从四段式 `api.yaml` 生成 AIP 风格的服务层 proto（gRPC）并编译成 `*.pb.go` / `*_grpc.pb.go`。

### 核心能力（P0 范围）

1. **四段式 YAML 解析**：`import_protos`（依赖声明）/ `settings`（生成配置）/ `entities`（实体建模）/ `services`（服务组装）
2. **实体级方法生成**：`Create<Entity>`（各资源可选，部分创建，只返回 key）/ `Delete<Entity>`（硬删除）/ `Delete<Entity>Soft`（软删除，可与硬删除共存）
3. **资源级方法生成**：`Get<Entity><Resource>` / `BatchGet<Entity><Resource>s` / `List<Entity><Resource>s`（分页/过滤/排序）/ `Update<Entity><Resource>`（update_mask 部分更新）
4. **版本/乐观锁**：STRONG（标量 CAS）/ WEAK（`google.protobuf.*Value` wrapper，可区分未设置）/ NONE
5. **依赖管理三路径**：
   - `path`（本地 proto 源）→ protocompile `SourceResolver` 直接解析
   - `git`（远程仓库）→ `go-git` library clone（无 buf CLI 依赖）
   - `bsr`（BSR 模块）→ subprocess 调用 buf CLI（`buf dep update` + `buf export`）
6. **双锁机制**：`api.lock`（git 依赖，apigen 自管）+ `buf.lock`/`buf.yaml`（BSR 依赖，buf CLI 自管）
7. **通用 option 注入**：field/message/rpc/service/file 五档纯搬运，工具不解析语义
8. **编译成 stub**：subprocess 调用 `protoc-gen-go` / `protoc-gen-go-grpc`，无需 protoc 二进制
9. **后置校验**：api-linter（带行内豁免）+ buf breaking（可选，仅 BSR 依赖时）
10. **可编译性保证**：四道防线（复合 Resolver 统一解析 → 编译前闭包 dry-run → 传递依赖递归收全 → generate/build 复用同一 FDSet）

### CLI 子命令

- `apigen generate`：校验 → 拉取依赖 → 生成 proto
- `apigen build`：generate + 编译成 pb.go/grpc.pb.go
- `apigen dep update`：强制刷新远程依赖
- `apigen dep prune`：文件级反查移除未引用依赖
- `apigen entity list`：干跑预览

### 不在本次范围

- HTTP 转码（`google.api.http` 注解 + grpc-gateway）→ 后续提案
- OpenAPI/swagger 生成 → 后续提案
- JS stub 生成（es/connect-es）→ 后续提案
- einride/aip-go 插件 → 后续提案

## Impact

### 新增代码

- `cmd/apigen/`：CLI 入口（generate/build/dep/entity 子命令）
- `internal/yaml/`：api.yaml schema 解析与校验
- `internal/dep/`：三路径依赖拉取（path/git/bsr）+ 锁文件管理
- `internal/parse/`：protocompile 复合 Resolver 封装
- `internal/ir/`：IR 构建（实体→资源→方法映射，wrapper 字段号分配，option 注入）
- `internal/render/`：text/template 模板渲染生成 proto
- `internal/build/`：subprocess 调用 protoc-gen-* 插件
- `internal/lint/`：api-linter 豁免生成与调用
- `internal/cli/`：CLI 框架（cobra/urfave）

### 依赖

- `bufbuild/protocompile`（library，proto 解析）
- `go-git/go-git`（library，git 依赖拉取）
- `buf CLI`（subprocess，仅 BSR 依赖时）
- `protoc-gen-go` / `protoc-gen-go-grpc`（subprocess，编译 stub）
- `api-linter`（subprocess，后置校验）

### 用户工作区产物

- `api.yaml`（用户编写）
- `api.lock`（apigen 生成，git 依赖时）
- `buf.yaml` / `buf.lock`（apigen/buf CLI 生成，仅 BSR 依赖时）
- `generated/proto/<service>/<service>.proto`（生成）
- `generated/go/<service>/*.pb.go` / `*_grpc.pb.go`（生成）
