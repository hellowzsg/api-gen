## ADDED Requirements

### Requirement: JS Plugin Declaration

apigen SHALL support declaring JS stub generation via `settings.plugins.js` field in `api.yaml`. The field is a string array; when it contains `"es"`, apigen enables `protoc-gen-es` invocation during `apigen build`.

#### Scenario: plugins.js omitted (backward compatibility)
- **WHEN** `api.yaml` has no `settings.plugins` section or `settings.plugins.js` is omitted/empty
- **THEN** `apigen build` does NOT invoke `protoc-gen-es` and produces no JS output, behaving identically to pre-this-change builds

#### Scenario: plugins.js declares es
- **WHEN** `api.yaml` declares `settings.plugins.js: [es]`
- **THEN** `apigen build` invokes `protoc-gen-es` subprocess and generates TypeScript files under `settings.out.js` (default `generated/js`)

#### Scenario: plugins.js declares unknown plugin
- **WHEN** `api.yaml` declares `settings.plugins.js: [unknown]` with a value other than `"es"`
- **THEN** `apigen build` fails fast with an error message indicating the unknown JS plugin name

### Requirement: protoc-gen-es Invocation

When JS stub generation is enabled, apigen SHALL invoke `protoc-gen-es` via the standard stdin/stdout `CodeGeneratorRequest`/`CodeGeneratorResponse` protocol, reusing the same `CodeGeneratorRequest` built for Go plugins (complete transitive closure of proto files).

#### Scenario: protoc-gen-es generates TypeScript files
- **WHEN** `apigen build` runs with `plugins.js: [es]` and `protoc-gen-es` is installed in PATH
- **THEN** TypeScript files (`.ts`) are generated under `settings.out.js`, with file paths following proto-relative layout (e.g. `generated/js/demo/business/book/book_pb.ts`)

#### Scenario: protoc-gen-es plugin parameter
- **WHEN** `apigen build` invokes `protoc-gen-es`
- **THEN** the `CodeGeneratorRequest.parameter` is set to `target=ts` (TypeScript output)

#### Scenario: protoc-gen-es not installed
- **WHEN** `plugins.js: [es]` is declared but `protoc-gen-es` binary is not found in PATH
- **THEN** `apigen build` fails fast with an error indicating the plugin is missing and suggesting `go install`

### Requirement: JS Output Directory Derivation

apigen SHALL derive the JS output directory from `settings.out.js` (default `generated/js` when empty). The directory is created if it does not exist.

#### Scenario: out.js explicitly set
- **WHEN** `api.yaml` declares `settings.out.js: custom/js`
- **THEN** `protoc-gen-es` output is written under `custom/js/`

#### Scenario: out.js omitted
- **WHEN** `api.yaml` omits `settings.out.js`
- **THEN** `protoc-gen-es` output is written under `generated/js/` (default)

### Requirement: CLI Build Integration

The `internal/cli/build.go` SHALL read `cfg.Settings.Plugins.JS`, determine whether `"es"` is declared (setting `generateJS = true`), derive `jsOutDir` from `cfg.Settings.Out.Js`, and pass both to `build.Compile`.

#### Scenario: build passes jsOutDir and generateJS to Compile
- **WHEN** `apigen build` runs with `plugins.js: [es]` and `out.js: generated/js`
- **THEN** `build.Compile` is called with `jsOutDir = "generated/js"` and `generateJS = true`

#### Scenario: build omits JS when plugins.js empty
- **WHEN** `apigen build` runs without `plugins.js` declaration
- **THEN** `build.Compile` is called with `generateJS = false` (and `jsOutDir` may be empty), no JS directory is created
