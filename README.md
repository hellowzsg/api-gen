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

## 设计文档

详见 [design-v2.md](design-v2.md)。
