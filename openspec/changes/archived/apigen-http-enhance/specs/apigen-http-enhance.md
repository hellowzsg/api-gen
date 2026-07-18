## ADDED Requirements

### Requirement: OpenAPI v2 Generation

When `settings.http.enable: true` and `settings.http.generate_openapi: true`, apigen MUST invoke `protoc-gen-openapiv2` to generate `<service>.swagger.json` files into `settings.out.openapi` (default `generated/openapi`), one per service.

#### Scenario: generate_openapi enabled produces swagger.json
- **WHEN** `api.yaml` has `settings.http.enable: true` and `settings.http.generate_openapi: true`
- **AND** `apigen build` is executed
- **THEN** for each service, a `<service>.swagger.json` file is generated under `settings.out.openapi`
- **AND** the swagger.json contains paths matching the generated `google.api.http` annotations

#### Scenario: generate_openapi disabled produces no swagger
- **WHEN** `settings.http.enable: true` and `settings.http.generate_openapi` is false or unset
- **THEN** no swagger.json files are generated
- **AND** `protoc-gen-openapiv2` is not invoked

#### Scenario: HTTP disabled ignores generate_openapi
- **WHEN** `settings.http.enable` is false or `settings.http` is nil
- **AND** `settings.http.generate_openapi: true` is declared
- **THEN** no error is raised (generate_openapi is silently ignored)
- **AND** no swagger.json files are generated

### Requirement: Per-Method HTTP Override

`reader.http` and `writer.update.http` MAY override the default HTTP verb, path, body, and body_style for a method. Fields not specified in the `http` block inherit the global `settings.http` defaults.

#### Scenario: override verb and path for List
- **WHEN** a resource declares `reader.list: true` with `reader.http: { verb: get, path: /library/LibraryService/book/{key.id}/metadata }`
- **THEN** the generated `google.api.http` annotation for List uses `get: "/library/LibraryService/book/{key.id}/metadata"` with no body
- **AND** the default POST + body:"*" is not applied

#### Scenario: override body_style for Update
- **WHEN** a resource declares `writer.update.http: { body_style: resource }`
- **AND** the resource name is `meta`
- **THEN** the generated `google.api.http` annotation for Update uses `body: "meta"`

#### Scenario: partial override inherits unspecified fields
- **WHEN** a resource declares `writer.update.http: { body_style: resource }` without specifying `verb` or `path`
- **THEN** the verb defaults to PATCH and the path defaults to the standard flat path

#### Scenario: user-written path variable validated against key leaves
- **WHEN** a user writes `http.path: /library/LibraryService/book/{key.org.oid}/meta`
- **AND** the key type has a scalar leaf at dot-path `org.oid`
- **THEN** the path is accepted
- **WHEN** the user writes `{key.nonexistent}` and no such leaf exists
- **THEN** apigen fails fast with a clear error indicating the unreachable path variable

### Requirement: Custom Method HTTP Routing

`custom_methods[].http` MAY declare an HTTP route using AIP-136 colon syntax for custom verbs.

#### Scenario: custom method with colon syntax
- **WHEN** a service declares a custom method `ArchiveBook` with `http: { verb: post, path: /library/LibraryService/book/{book_id}:archive, body: "*" }`
- **THEN** the generated RPC has `option (google.api.http) = { post: "/library/LibraryService/book/{book_id}:archive" body: "*" }`
- **AND** the custom method RPC is rendered with the HTTP annotation inside the RPC body

#### Scenario: custom method without http block
- **WHEN** a custom method is declared without an `http` block
- **THEN** the RPC is rendered as a plain gRPC method with no `google.api.http` annotation

#### Scenario: custom method path variable syntax validation
- **WHEN** a custom method declares `http.path: /library/LibraryService/book/{book_id}:archive`
- **THEN** apigen validates the path variable syntax is well-formed (non-empty, valid characters)
- **AND** apigen does NOT recursively resolve the variable against the custom request message (field existence is deferred to gateway plugin compile time)

### Requirement: body_style: resource

`settings.http.body_style: resource` (global) or per-method `http.body_style: resource` binds the HTTP body to the resource field name instead of the whole wrapper.

#### Scenario: global body_style resource for Update
- **WHEN** `settings.http.body_style: resource` is set globally
- **AND** a resource named `meta` has an Update method
- **THEN** the generated `google.api.http` annotation uses `body: "meta"`

#### Scenario: body_style resource rejected for multi-resource Create
- **WHEN** `settings.http.body_style: resource` is set globally
- **AND** an entity has multiple resources in its Create request
- **THEN** apigen fails fast with an error indicating body_style: resource is ambiguous for multi-resource Create
- **AND** suggests using body_style: wrapper (default) for Create

#### Scenario: per-method body_style overrides global
- **WHEN** `settings.http.body_style: wrapper` is set globally
- **AND** a specific Update method declares `http: { body_style: resource }`
- **THEN** only that Update method uses `body: "<resource field>"`, all other methods use the global default

### Requirement: api-linter Exemption Adjustment

The `core::0133::http-body` exemption for Create MUST NOT be emitted when `body_style: resource` is in effect for Create (body is a resource field, not `*`).

#### Scenario: body_style wrapper emits http-body exemption
- **WHEN** Create uses `body_style: wrapper` (default, `body: "*"`)
- **THEN** the `core::0133::http-body` exemption is emitted

#### Scenario: body_style resource suppresses http-body exemption
- **WHEN** Create uses `body_style: resource` (body is a resource field)
- **THEN** the `core::0133::http-body` exemption is NOT emitted for that Create

### Requirement: OpenAPI Output Directory

When `settings.out.openapi` is declared, swagger.json files are written there; otherwise the default `generated/openapi` is used.

#### Scenario: custom openapi output directory
- **WHEN** `settings.out.openapi: custom/openapi` is declared
- **AND** `generate_openapi: true`
- **THEN** swagger.json files are written to `<api.yaml dir>/custom/openapi/<service>.swagger.json`

#### Scenario: default openapi output directory
- **WHEN** `settings.out.openapi` is not declared
- **AND** `generate_openapi: true`
- **THEN** swagger.json files are written to `<api.yaml dir>/generated/openapi/<service>.swagger.json`
