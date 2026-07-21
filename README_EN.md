# apigen

> Generate AIP-style gRPC service definitions from an entity model, with on-demand Go, HTTP gateway, OpenAPI, and TypeScript output.

简体中文 | [English](README_EN.md)

## Overview

`apigen` is a declarative API generation tool. You describe your business entities—their keys, resource facets, read/write capabilities, concurrency controls, and service exposure—in a single `api.yaml`. The tool produces service-layer `.proto` files, standard Request/Response wrappers, pagination, update masks, and more, then optionally compiles them into multi-language client code.

## Feature Overview

- **Declarative definition**: Describe entities, resources, read/write policies, and services in `api.yaml`—no boilerplate proto needed.
- **Standard gRPC API generation**: Automatically generates AIP-compliant Create, Delete, Get, BatchGet, List, Update, and other methods.
- **Optimistic locking**: Built-in versioning strategies with CAS-based concurrent updates.
- **HTTP transcoding**: One-click generation of `google.api.http` annotations and grpc-gateway reverse-proxy code.
- **Multi-language clients**: Generate Go and TypeScript stubs, plus OpenAPI v2 API documentation.
- **Dependency management**: Reference proto types from local files, Git repositories, or BSR, with lockfile support for reproducible builds.

## Requirements

| Item | Requirement | When needed |
|---|---|---|
| Go | **1.24+** | Install and run `apigen` |
| `protoc-gen-go` | Required | Generate Go message code |
| `protoc-gen-go-grpc` | Required | Generate Go gRPC code |
| `protoc-gen-grpc-gateway` | Required when HTTP is enabled | Generate `*.pb.gw.go` |
| `protoc-gen-openapiv2` | Required for OpenAPI | Generate Swagger docs |
| `protoc-gen-es` | Required for TypeScript | Generate `*.pb.ts` |
| Git CLI | Required when using `import_protos.git` | Fetch Git proto dependencies |
| Buf CLI | Required when using `import_protos.bsr` | Export BSR modules |

> `apigen build` invokes generators via the Protobuf plugin protocol—**a standalone `protoc` installation is not required**. Make sure the necessary plugins are on your `PATH`.

## Installation & Quick Start

### 1. Install the CLI

```bash
git clone <repository-url> aip-gen
cd aip-gen
go install ./cmd/apigen

# Verify
apigen --help
```

### 2. Install the Required Go Plugins

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

If you need HTTP, OpenAPI, or TypeScript, install the corresponding plugins:

```bash
# HTTP transcoding & OpenAPI
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

# TypeScript
npm install --global @bufbuild/protoc-gen-es
```

### 3. Run the Built-in Example

The `examples/book/` directory demonstrates entities, resources, versioning, HTTP, OpenAPI, TypeScript, and service narrowing:

```bash
# Preview entities and methods without writing files
go run ./cmd/apigen entity list -f examples/book/api.yaml

# Generate service protos
go run ./cmd/apigen generate -f examples/book/api.yaml

# Generate protos and compile Go / HTTP / OpenAPI / TypeScript artifacts
go run ./cmd/apigen build -f examples/book/api.yaml
```

Generated files are placed under `examples/book/generated/`. See [examples/README.md](examples/README.md) for more details.

## Usage Example

### Define Your Domain Types

`apigen` generates the service layer; you maintain the entity key and resource messages. Below is a minimal example of `proto/demo/business/book/book.proto`:

```proto
syntax = "proto3";

package demo.business.book;

option go_package = "github.com/acme/demo-book/generated/go/demo/business/book;book";

message BookId {
  string id = 1;
}

message BookMeta {
  string title = 1;
  string author = 2;
}
```

### Write `api.yaml`

```yaml
syntax: v1
name: demo.business.book

import_protos:
  - path: "proto/**/*.proto"

settings:
  go_repo: github.com/acme/demo-book
  out:
    proto: generated/proto
    go: generated/go

entities:
  - name: book
    key: { type_: BookId }
    create: {}
    delete: {}
    resources:
      - name: meta
        type_: BookMeta
        version: { kind: STRONG, type: U64 }
        reader: { batch: true, list: true }
        writer:
          update: { mask: true }

services:
  - name: LibraryService
    entities:
      - name: book
```

### Generate & Compile

```bash
# Generate protos only
apigen generate -f api.yaml

# Generate protos and compile Go code
apigen build -f api.yaml

# List entities, resources, and methods
apigen entity list -f api.yaml
```

The above configuration generates `LibraryService` with methods including `CreateBook`, `DeleteBook`, `GetBookMeta`, `BatchGetBookMetas`, `ListBookMetas`, and `UpdateBookMeta`.

## Command Reference

All commands accept `--file` or `-f` to specify the config file, defaulting to `api.yaml` in the current directory.

| Command | Description |
|---|---|
| `apigen generate -f api.yaml` | Validate config and dependencies, then generate per-service `.proto` files. |
| `apigen build -f api.yaml` | Run `generate` first, then invoke installed code-generation plugins. |
| `apigen entity list -f api.yaml` | Print entities, resources, and generated methods without writing files. |
| `apigen dep update -f api.yaml` | Refresh remote dependencies and update Git entries in `api.lock`. |
| `apigen dep prune -f api.yaml` | Placeholder for dependency cleanup; currently does not remove existing lock entries. |

## `api.yaml` Configuration

### Structure

```yaml
syntax: v1
name: example.catalog

import_protos: []
settings: {}
entities: []
services: []
```

The config is parsed in strict-field mode—typos and unknown fields are rejected immediately. `entities` is required; each entity must define a key and at least one resource.

### Root Fields & Dependency Sources

| Field | Type | Purpose |
|---|---|---|
| `syntax` | `string` | Config format identifier. Example uses `v1`. |
| `name` | `string` | Business proto package in dot-separated form, e.g., `demo.business.book`. |
| `import_protos` | `[]object` | Declares where `key.type_` and resource `type_` protos are defined. |
| `settings` | `object` | Controls output paths, HTTP behavior, and language plugins. |
| `entities` | `[]object` | Defines business entities, resources, and read/write capabilities. |
| `services` | `[]object` | Defines the gRPC services to expose. |

Each `import_protos` entry chooses one of the following sources:

| Field | Description |
|---|---|
| `path` | Local proto glob, resolved relative to the `api.yaml` directory. |
| `git` | Git repository URL. Combine with `ref` (branch, tag, or commit) and `subdir` (proto subdirectory within the repo). |
| `bsr` | BSR module name, e.g., `buf.build/googleapis/googleapis`. |
| `alias` | Accepted compat field; not currently used for `type_` alias resolution. |
| `version` | Accepted compat field for BSR entries; the current export flow does not pin to this value. |

Example:

```yaml
import_protos:
  - path: "proto/**/*.proto"
  - git: https://github.com/googleapis/googleapis
    ref: master
    subdir: google
  - bsr: buf.build/googleapis/googleapis
```

> When HTTP is enabled, `google/api/annotations.proto` and its transitive imports must be resolvable from your dependencies (via local vendored proto, Git, or BSR).

### `settings` — Output & Generation

| Field | Type | Purpose |
|---|---|---|
| `go_repo` | `string` | Go module path written into `go_package` of generated protos. |
| `js_repo` | `string` | Accepted compat field; does not currently affect TypeScript output. |
| `out.proto` | `string` | Directory for generated service `.proto` files. |
| `out.go` | `string` | Go code output directory. |
| `out.js` | `string` | TypeScript output directory. |
| `out.openapi` | `string` | OpenAPI v2 output directory. |
| `http.enable` | `bool` | Enable `google.api.http` annotations and grpc-gateway code generation. |
| `http.prefix` | `string` | Global prefix for auto-generated HTTP routes, e.g., `/api`. |
| `http.body_style` | `string` | Default HTTP body strategy: `wrapper` (default, equivalent to `body: "*"`) or `resource`. |
| `http.generate_openapi` | `bool` | Whether to generate OpenAPI v2 docs. Only effective when HTTP is enabled and `out.openapi` is set. |
| `plugins.js` | `[]string` | JavaScript plugin list; currently only `es` is supported. |

```yaml
settings:
  go_repo: github.com/acme/demo-book
  out:
    proto: generated/proto
    go: generated/go
    js: generated/js
    openapi: generated/openapi
  http:
    enable: true
    prefix: /library
    body_style: wrapper
    generate_openapi: true
  plugins:
    js: [es]
```

### `entities` — Domain Model & Operations

An entity consists of a key and one or more resources. A resource represents an independent data facet of the same business object—for example, a Book's `meta` (metadata) and `content` (body). Both `key.type_` and resource `type_` must reference imported, developer-maintained proto messages.

#### Entity Fields

| Field | Type | Purpose |
|---|---|---|
| `name` | `string` | Entity name in `snake_case`, used as the naming stem for generated types and methods. |
| `key.type_` | `string` | Key message type; can use a fully-qualified name. |
| `create` | `object` | Set to `{}` to generate `Create`, whose response carries only the key. |
| `delete` | `object` | Set to `{}` to generate a hard-delete `Delete`. |
| `delete_soft` | `object` | Set to `{}` to generate a soft-delete `DeleteSoft`; can coexist with `delete`. |
| `resources` | `[]object` | At least one resource is required. |

#### Resource Fields

| Field | Type | Purpose |
|---|---|---|
| `name` | `string` | Resource name in `snake_case`. |
| `type_` | `string` | Resource message type; can use a fully-qualified name. |
| `version.kind` | `string` | Update concurrency control: `STRONG`, `WEAK`, or `NONE`. |
| `version.type` | `string` | Version value type: `U64`, `U32`, or `STRING`; required for `STRONG` and `WEAK`. |
| `reader` | `object` | Read capabilities. `reader: {}` generates `Get`. |
| `reader.batch` | `bool` | Generate `BatchGet`. |
| `reader.list` | `bool` | Generate `List` with pagination, `filter`, and `order_by` fields. |
| `reader.list_config.total_size` | `bool` | Include `total_size` in List response; included when omitted or set to `true`. |
| `reader.http` | `object` | Overrides only `List` HTTP `verb`, `path`, `body`, or `body_style`. |
| `writer.update` | `object` | Configure to generate `Update`. |
| `writer.update.mask` | `bool` | Include a `google.protobuf.FieldMask`-typed `update_mask` in Update requests. |
| `writer.update.http` | `object` | Override Update HTTP `verb`, `path`, `body`, or `body_style`. |
| `options` | `[]object` | Parsed and validated reserved config; not currently written into generated proto options. |

#### Version Strategies

| Strategy | Get Response | Update Request | Update Response | Use Case |
|---|---|---|---|---|
| `STRONG` | Returns scalar `version` | Must carry scalar `version` for CAS | Returns updated scalar version | Enforce CAS to prevent concurrent overwrites |
| `WEAK` | Returns wrapper-type `version` | Optionally carries wrapper-type `version` | Returns updated wrapper-type version | Allow clients to choose whether to perform CAS |
| `NONE` | No version returned | No version carried | Returns `google.protobuf.Empty` | Direct updates without optimistic locking |

For `STRONG` the scalar type, and for `WEAK` the corresponding wrapper type, is determined by `version.type`: `U64`, `U32`, or `STRING`.

### `services` — Service Exposure & Custom Methods

A service can expose an entity's full capabilities, or narrow them to a subset of resources and methods.

| Field | Type | Purpose |
|---|---|---|
| `services[].name` | `string` | Service name in `PascalCase`, e.g., `LibraryService`. |
| `services[].entities[].name` | `string` | References an entity defined in `entities`. |
| `services[].entities[].resources` | `[]object` | Optional narrowing rules; currently uses resource `name`, `reader.batch`, `reader.list`, and `writer.update`. Omitting inherits the entity's full capabilities. |
| `services[].custom_methods` | `[]object` | Service-level custom RPC list. |
| `custom_methods[].name` | `string` | Custom RPC name in `PascalCase`. |
| `custom_methods[].request` | `string` | Request message type. |
| `custom_methods[].response` | `string` | Response message type. |
| `custom_methods[].http` | `object` | When HTTP is enabled, set `verb`, `path`, and `body`; paths support AIP-136 colon syntax. |

```yaml
services:
  - name: LibraryService
    entities:
      - name: book                 # Expose all of book's capabilities

  - name: AdminService
    entities:
      - name: book
        resources:
          - name: meta
            reader: { list: true } # Only expose ListBookMetas
    custom_methods:
      - name: ArchiveBook
        request: ArchiveBookRequest
        response: ArchiveBookResponse
        http:
          verb: post
          path: /library/books/{book_id}:archive
          body: "*"
```

## Project Structure

```text
aip-gen/
├── cmd/apigen/        # CLI entry point
├── internal/
│   ├── cli/           # Cobra commands
│   ├── yaml/          # api.yaml parsing & validation
│   ├── dep/           # path / Git / BSR dependency resolution
│   ├── ir/            # Intermediate representation & entity modeling
│   ├── render/        # Proto & HTTP annotation rendering
│   └── build/         # Protobuf plugin orchestration
├── examples/book/     # End-to-end example
├── openspec/          # Design change records
└── design-v2.md       # Architecture design & roadmap
```

## License

The project is declared under the **MIT License**. A `LICENSE` file is not yet present in the repository; please add an official license file before publishing, redistributing, or consuming as a dependency.
