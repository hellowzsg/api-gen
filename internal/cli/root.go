// Package cli implements the apigen command-line interface.
package cli

import (
	"github.com/spf13/cobra"
)

// Version is the CLI version, injected at release build time via
// -ldflags "-X github.com/hellowzsg/api-gen/internal/cli.Version=vX.Y.Z".
var Version = "dev"

// NewRoot creates the root apigen command.
func NewRoot() *cobra.Command {
	var verbose bool
	root := &cobra.Command{
		Use:     "apigen",
		Short:   "AIP Proto 标准化生成工具",
		Version: Version,
		Long: "apigen 从四段式 api.yaml 生成 AIP 风格的服务层 proto（gRPC），" +
			"并一键编译成 *.pb.go / *_grpc.pb.go。",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initLogging(verbose)
		},
	}
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "输出关键节点结构化日志（stderr）")
	root.AddCommand(newGenerateCmd(), newBuildCmd(), newDepCmd(), newEntityCmd())
	return root
}

func newGenerateCmd() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "generate",
		Short: "校验 → 拉取依赖 → 生成 proto",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(cmd.Context(), file)
		},
	}
	c.Flags().StringVarP(&file, "file", "f", "api.yaml", "path to api.yaml")
	return c
}

func newBuildCmd() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "build",
		Short: "generate + 编译成 pb.go/grpc.pb.go",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(cmd.Context(), file)
		},
	}
	c.Flags().StringVarP(&file, "file", "f", "api.yaml", "path to api.yaml")
	return c
}

func newDepCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "dep",
		Short: "依赖管理",
	}
	c.AddCommand(newDepUpdateCmd(), newDepPruneCmd())
	return c
}

func newDepUpdateCmd() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "update",
		Short: "强制重新拉取所有远程依赖",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDepUpdate(cmd.Context(), file)
		},
	}
	c.Flags().StringVarP(&file, "file", "f", "api.yaml", "path to api.yaml")
	return c
}

func newDepPruneCmd() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "prune",
		Short: "移除未被引用的远程依赖",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDepPrune(cmd.Context(), file)
		},
	}
	c.Flags().StringVarP(&file, "file", "f", "api.yaml", "path to api.yaml")
	return c
}

func newEntityCmd() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "entity",
		Short: "实体管理",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "干跑预览：列出所有实体、资源、方法清单",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEntityList(cmd.Context(), file)
		},
	}
	list.Flags().StringVarP(&file, "file", "f", "api.yaml", "path to api.yaml")
	c.AddCommand(list)
	return c
}
