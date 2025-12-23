# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
go build ./...              # Build provider binary
go fmt ./...                # Format code
go vet ./...                # Static analysis
go mod tidy                 # Update dependencies
make all                    # Format, vet, and build
```

## Testing with Existing Habitica Data

Import existing Habitica resources:
```bash
# Generate imports.tf and resources.tf from your Habitica account
make generate-imports

# Test locally (requires .terraformrc with dev_overrides)
cd examples/import
terraform init
terraform plan
terraform apply
```

## Local Development Setup

Build and install locally for Terraform to use:
```bash
go build -o terraform-provider-habitica

# Install to local plugin directory
VERSION=0.2.0
PLUGIN_DIR=~/.terraform.d/plugins/registry.terraform.io/inannamalick/habitica/${VERSION}/darwin_arm64
mkdir -p $PLUGIN_DIR
cp terraform-provider-habitica $PLUGIN_DIR/terraform-provider-habitica_v${VERSION}
```

Or use `.terraformrc` dev overrides (see `.terraformrc.example`).

## Releases

Tagged releases trigger GitHub Actions to build multi-platform binaries via GoReleaser:
```bash
git tag v0.x.0
git push origin v0.x.0
# Binaries published at github.com/inanna-malick/habitica-terraform/releases
```

## Architecture

This is a Terraform provider for Habitica built with `terraform-plugin-framework`.

### Layer Structure

```
main.go                              # Provider server entry point
internal/
├── provider/provider.go             # Provider config, auth, resource/datasource registration
├── client/
│   ├── client.go                    # HTTP client with rate limiting, retries, caching
│   └── types.go                     # API types: Tag, Task, Webhook, RepeatConfig
├── resources/{tag,habit,daily,webhook}/
│   └── resource.go                  # CRUD implementation + schema
└── datasources/user_tasks/
    └── datasource.go                # Read-only data source for all tasks
```

### Data Flow

1. **Provider** (`internal/provider/`) configures auth from config or env vars, creates `*client.Client`
2. **Client** (`internal/client/`) passed to resources/datasources via `ProviderData`
3. **Resources** (`internal/resources/`) implement `resource.Resource` interface, call client methods
4. **Data Sources** (`internal/datasources/`) implement `datasource.DataSource` interface

### Rate Limiting & Caching

The Habitica API enforces 30 requests/60 seconds. The client handles this with:
- Rate limit tracking via `X-RateLimit-Remaining` header
- Pauses when remaining < buffer (default 5)
- Exponential backoff on 429 responses
- Bulk-fetch caching for `GetTask` and `GetTag` (one API call populates entire cache)
- Cache invalidation on create/update/delete operations

### Adding a New Resource

1. Create `internal/resources/<name>/resource.go`
2. Define model struct with `tfsdk` tags
3. Implement `Metadata`, `Schema`, `Configure`, `Create`, `Read`, `Update`, `Delete`
4. Add `ImportState` method for import support
5. Add client methods in `internal/client/client.go`
6. Register in `provider.Resources()` in `internal/provider/provider.go`

### Adding a New Data Source

1. Create `internal/datasources/<name>/datasource.go`
2. Define model struct with `tfsdk` tags (typically just output fields)
3. Implement `Metadata`, `Schema`, `Configure`, `Read`
4. Add client methods in `internal/client/client.go` if needed
5. Register in `provider.DataSources()` in `internal/provider/provider.go`

### API Mapping

| Resource/Data Source | API Endpoint | Notes |
|---------------------|--------------|-------|
| `habitica_tag` | `/api/v3/tags` | Simple name-only resource |
| `habitica_habit` | `/api/v3/tasks/user` | Task with `type: "habit"` |
| `habitica_daily` | `/api/v3/tasks/user` | Task with `type: "daily"`, complex repeat config |
| `habitica_webhook` | `/api/v3/user/webhook` | No single-get endpoint, uses list+filter |
| `habitica_user_tasks` | `/api/v3/tasks/user` + `/api/v3/tags` | Data source, resolves tag UUIDs to names |

### Data Source Output Format

`habitica_user_tasks` outputs JSON with resolved tag names:
```json
{
  "dailies": [{"id": "...", "text": "...", "completed": false, "isDue": true, "tags": ["tier:foundation"], "streak": 5}],
  "habits": [{"id": "...", "text": "...", "counterUp": 3, "counterDown": 0, "tags": ["exercise"]}],
  "todos": [{"id": "...", "text": "...", "completed": false, "tags": []}]
}
```

### Environment Variables

- `HABITICA_USER_ID` - User's UUID
- `HABITICA_API_TOKEN` - API token (sensitive)
- `HABITICA_CLIENT_AUTHOR_ID` - For x-client header (required by Habitica API)

### Import Generation

`scripts/generate_imports.py` fetches all tasks and tags, generates two files:
- `examples/import/imports.tf` - Temporary import blocks (delete after `terraform apply`)
- `examples/import/resources.tf` - Permanent resource definitions with tag references

## Critical Schema Patterns

### Avoid Computed+Default Combination

**DO NOT** use `Computed: true` with `Default:` on the same attribute. This causes "Value Conversion Error" when Terraform reconciles config vs state.

**Bad:**
```go
"up": schema.BoolAttribute{
    Optional: true,
    Computed: true,  // ❌ Don't combine with Default
    Default:  booldefault.StaticBool(true),
}
```

**Good:**
```go
"up": schema.BoolAttribute{
    Description: "Defaults to true if not specified.",
    Optional:    true,  // ✅ Just Optional, handle defaults in code
}

// In Create/Update methods:
up := getBoolWithDefault(plan.Up, true)
```

This pattern caused bugs in:
- `habitica_daily.repeat` nested attributes (fixed in v0.2.1)
- `habitica_habit.up` and `habitica_habit.down` (fixed in v0.2.2)

### Helper Pattern for Defaults

When using Optional-only attributes with defaults:

```go
func getBoolWithDefault(val types.Bool, defaultVal bool) bool {
    if val.IsNull() || val.IsUnknown() {
        return defaultVal
    }
    return val.ValueBool()
}
```

Use in Create/Update: `field := getBoolWithDefault(plan.Field, true)`

### Nested Attributes

For `SingleNestedAttribute`:
- Make parent attribute `Optional: true` (no Computed)
- Make child attributes `Optional: true` (no Computed, no Default)
- Handle defaults in `modelToTask` conversion functions
- Always populate from API response in `updateModelFromTask`

## Version History

- **v0.1.0** - Initial release with resources and import support
- **v0.2.0** - Added `habitica_user_tasks` data source
- **v0.2.1** - Fixed repeat field value conversion errors in dailies
- **v0.2.2** - Fixed up/down field value conversion errors in habits
