package cli

import (
	"context"
	"fmt"
)

// runGenerate 执行 generate 子命令：校验 → 拉取依赖 → 生成 proto。
func runGenerate(_ context.Context, _ string) error {
	return fmt.Errorf("generate not implemented yet")
}

// runBuild 执行 build 子命令：generate + 编译成 pb.go/grpc.pb.go。
func runBuild(_ context.Context, _ string) error {
	return fmt.Errorf("build not implemented yet")
}

// runDepUpdate 执行 dep update 子命令：强制重新拉取所有远程依赖。
func runDepUpdate(_ context.Context, _ string) error {
	return fmt.Errorf("dep update not implemented yet")
}

// runDepPrune 执行 dep prune 子命令：移除未被引用的远程依赖。
func runDepPrune(_ context.Context, _ string) error {
	return fmt.Errorf("dep prune not implemented yet")
}

// runEntityList 执行 entity list 子命令：干跑预览实体/资源/方法清单。
func runEntityList(_ context.Context, _ string) error {
	return fmt.Errorf("entity list not implemented yet")
}
