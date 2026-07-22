## ADDED Requirements

### Requirement: List filter_type Configuration

When a resource declares `reader.list: true` with `reader.list_config.filter_type` set to a non-empty value, the generated List request message MUST use the specified type as the `filter` field's type instead of the default `string`.

#### Scenario: filter_type declared uses custom message type

- **WHEN** a resource declares `reader.list: true` and `reader.list_config.filter_type: BookMetaFilter`
- **AND** `apigen build` is executed
- **THEN** the generated `List<Entity><Resource>sRequest` message contains `BookMetaFilter filter = 3;` instead of `string filter = 3;`

#### Scenario: filter_type omitted defaults to string

- **WHEN** a resource declares `reader.list: true` without `filter_type` (or `filter_type` is empty)
- **THEN** the generated List request message contains `string filter = 3;` (fully backward compatible)

#### Scenario: filter_type validated as syntactically valid type name

- **WHEN** a resource declares `reader.list_config.filter_type` with an empty string or a value starting with `.` or a digit
- **THEN** apigen fails fast with a clear error indicating the invalid filter_type value

#### Scenario: filter_type does not affect field number

- **WHEN** `filter_type` is set to any value
- **THEN** the `filter` field number remains 3 (unchanged from the default)

### Requirement: List Default HTTP Verb

When a resource declares `reader.list: true` without a `reader.http` override, the generated List method MUST use POST with `body: "*"` as the default HTTP annotation, regardless of whether `filter_type` is declared.

#### Scenario: List with custom filter_type uses POST by default

- **WHEN** a resource declares `reader.list: true` and `reader.list_config.filter_type: BookMetaFilter`
- **AND** no `reader.http` override is declared
- **THEN** the generated `google.api.http` annotation uses `post: "/<prefix>/<svc>/<entity>/<resource>/list"` with `body: "*"`

### Requirement: filter_type Reachability

The `filter_type` value MUST be a syntactically valid proto type name. Type reachability (the message exists in the import closure) is deferred to the protocompile link phase, consistent with `custom_method.request`/`response` handling.

#### Scenario: filter_type unreachable fails at compile time

- **WHEN** `filter_type` is set to `NonExistentFilter`
- **AND** no such message exists in the import closure
- **THEN** apigen build fails at the protocompile link phase with an error referencing the unreachable type
