package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/acme/apigen/internal/dep"
	"github.com/acme/apigen/internal/ir"
	"github.com/acme/apigen/internal/render"
	apigenyaml "github.com/acme/apigen/internal/yaml"
)

func runGenerate(ctx context.Context, apiYAMLPath string) error {
	p, err := Prepare(ctx, apiYAMLPath)
	if err != nil {
		return err
	}
	return renderServiceProtos(p)
}

// renderServiceProtos renders all service protos into a staging directory
// first; only swaps into the real output dir once every service has rendered
// successfully. This makes generate atomic — a failure midway leaves the
// previous output intact.
func renderServiceProtos(p *Pipeline) error {
	protoOutDir := filepath.Join(p.BaseDir, p.Config.Settings.Out.Proto)
	staging, err := newStagingDir(protoOutDir)
	if err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}
	// Ensure staging is cleaned up if we return before committing.
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()
	for _, svc := range p.IR.Services {
		output, err := render.RenderServiceProto(p.IR, svc)
		if err != nil {
			return fmt.Errorf("render service %s: %w", svc.Name, err)
		}
		outPath := filepath.Join(staging, ir.ToSnakeCase(svc.Name), ir.ToSnakeCase(svc.Name)+".proto")
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
		if err := os.WriteFile(outPath, []byte(output), 0644); err != nil {
			return fmt.Errorf("write proto file: %w", err)
		}
	}
	slog.Info("proto rendered", "services", len(p.IR.Services))
	if err := commitDir(staging, protoOutDir); err != nil {
		return fmt.Errorf("commit proto output: %w", err)
	}
	slog.Info("output committed", "dir", protoOutDir)
	committed = true
	return nil
}

func parseConfig(path string) (*apigenyaml.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return apigenyaml.Parse(f)
}

// buildIR constructs the IR from the YAML config, optionally enriching it
// with key type descriptors for HTTP path binding. When HTTP is disabled,
// behavior is identical to ir.Build (P0). When HTTP is enabled, key type
// descriptors are fetched from the CompositeResolver so that ExtractKeyLeaves
// can run.
func buildIR(cfg *apigenyaml.Config, cr *dep.CompositeResolver) (*ir.IR, error) {
	httpEnabled := cfg.Settings.HTTP != nil && cfg.Settings.HTTP.Enable
	if !httpEnabled {
		return ir.Build(cfg)
	}
	// Build KeyDescriptors map for HTTP key-leaf extraction.
	keyDescs := make(map[string]protoreflect.MessageDescriptor, len(cfg.Entities))
	for _, e := range cfg.Entities {
		keyType := cfg.ResolveTypeName(e.Key.Type)
		md := cr.FindMessageDescriptor(keyType)
		if md == nil {
			return nil, fmt.Errorf("HTTP enabled but key type %q descriptor not found in resolved protos", keyType)
		}
		keyDescs[keyType] = md
	}
	return ir.BuildWithOptions(cfg, ir.BuildOptions{KeyDescriptors: keyDescs})
}

func validateTypeReferences(cfg *apigenyaml.Config, cr *dep.CompositeResolver) error {
	for _, e := range cfg.Entities {
		keyType := cfg.ResolveTypeName(e.Key.Type)
		if err := cr.CheckTypeIsMessage(keyType); err != nil {
			return fmt.Errorf("entity %q key.type_ %q: %w", e.Name, keyType, err)
		}
		for _, r := range e.Resources {
			resType := cfg.ResolveTypeName(r.Type)
			if err := cr.CheckTypeIsMessage(resType); err != nil {
				return fmt.Errorf("entity %q resource %q type_ %q: %w", e.Name, r.Name, resType, err)
			}
		}
	}
	return nil
}


