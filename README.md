# apigen

AIP Proto 标准化生成工具。

## 概述

apigen 从四段式 `api.yaml` 生成 AIP 风格的服务层 proto（gRPC），并一键编译成 `*.pb.go` / `*_grpc.pb.go`。

## 安装

```bash
go install github.com/acme/apigen/cmd/apigen@latest
```

## 使用

```bash
# 生成 proto
apigen generate -f api.yaml

# 生成 + 编译成 Go stub
apigen build -f api.yaml

# 强制刷新远程依赖
apigen dep update -f api.yaml

# 移除未引用的远程依赖
apigen dep prune -f api.yaml

# 预览实体/资源/方法清单
apigen entity list -f api.yaml
```

## 示例

`examples/book/` 是一个完整的端到端示例，演示：

- 四段式 `api.yaml`（含 path 依赖、两个 service、实体级 + 资源级方法、STRONG/NONE version）
- `LibraryService`（全量继承实体能力）
- `AdminService`（收窄到仅 `meta` 资源的 `list` reader 方法）
- 生成的 proto + Go stub 可直接 `go build`

```bash
# 从仓库根目录运行
go run ./cmd/apigen build -f examples/book/api.yaml

# 验证生成的 Go 代码可编译
cd examples/book && go build ./...
```

生成产物：
- `generated/proto/<service>/<service>.proto` — AIP 风格服务 proto
- `generated/go/<service>/<service>.pb.go` / `*_grpc.pb.go` — Go gRPC stub
- `generated/go/<type_proto_rel_path>/<type>.pb.go` — 用户 type_ proto 编译产物

## 设计文档

详见 [design-v2.md](design-v2.md)。
