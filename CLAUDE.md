# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
go build ./...              # Build provider binary
go fmt ./...                # Format code
go vet ./...                # Static analysis
go mod tidy                 # Update dependencies
```

## Local Testing

Build and install locally for Terraform to use:
```bash
go build -o terraform-provider-habitica
# Move to ~/.terraform.d/plugins/registry.terraform.io/inannamalick/habitica/0.1.0/darwin_arm64/
```

## Architecture

This is a Terraform provider for Habitica built with `terraform-plugin-framework`.

### Layer Structure

```
main.go                           # Provider server entry point
internal/
├── provider/provider.go          # Provider config, auth, resource registration
├── client/
│   ├── client.go                 # HTTP client with rate limiting & retries
│   └── types.go                  # API types: Tag, Task, Webhook, RepeatConfig
└── resources/{tag,habit,daily,webhook}/
    └── resource.go               # CRUD implementation + schema
```

### Data Flow

1. **Provider** (`internal/provider/`) configures auth from config or env vars, creates `*client.Client`
2. **Client** (`internal/client/`) passed to resources via `ResourceData`
3. **Resources** (`internal/resources/`) implement `resource.Resource` interface, call client methods

### Rate Limiting

The Habitica API enforces 30 requests/60 seconds. The client handles this automatically:
- Tracks `X-RateLimit-Remaining` header
- Pauses when remaining < buffer (default 5)
- Exponential backoff on 429 responses

### Adding a New Resource

1. Create `internal/resources/<name>/resource.go`
2. Define model struct with `tfsdk` tags
3. Implement `Metadata`, `Schema`, `Configure`, `Create`, `Read`, `Update`, `Delete`
4. Add client methods in `internal/client/client.go`
5. Register in `provider.Resources()` in `internal/provider/provider.go`

### API Mapping

| Resource | API Endpoint | Notes |
|----------|--------------|-------|
| `habitica_tag` | `/api/v3/tags` | Simple name-only resource |
| `habitica_habit` | `/api/v3/tasks/user` | Task with `type: "habit"` |
| `habitica_daily` | `/api/v3/tasks/user` | Task with `type: "daily"`, complex repeat config |
| `habitica_webhook` | `/api/v3/user/webhook` | No single-get endpoint, uses list+filter |

### Environment Variables

- `HABITICA_USER_ID` - User's UUID
- `HABITICA_API_TOKEN` - API token (sensitive)
- `HABITICA_CLIENT_AUTHOR_ID` - For x-client header (required by Habitica API)
