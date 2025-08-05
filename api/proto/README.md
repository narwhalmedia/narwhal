# Protocol Buffers API

This directory contains the Protocol Buffer definitions for the Narwhal media platform.

## Structure

```tree
api/proto/
├── common/v1/      # Common types shared across services
├── library/v1/     # Library service definitions
├── streaming/v1/   # Streaming service definitions
├── auth/v1/        # Authentication service definitions
├── acquisition/v1/ # Content acquisition service definitions
└── README.md       # This file
```

## Buf Configuration

This project uses [Buf](https://buf.build) for Protocol Buffer management.

### Configuration Files

- **`buf.yaml`**: Main Buf configuration with lint and breaking change rules
- **`buf.gen.yaml`**: Code generation configuration
- **`buf.work.yaml`**: Workspace configuration

### Common Commands

```bash
# Lint proto files
buf lint

# Check for breaking changes
buf breaking --against '.git#branch=main'

# Generate code
buf generate

# Format proto files
buf format -w

# Push to Buf Schema Registry (if configured)
buf push
```

### Code Generation

The project is configured to generate:

- Go Protocol Buffer code
- Go gRPC service code
- gRPC-Gateway REST proxy
- OpenAPI documentation

To regenerate code:

```bash
buf generate
```

Or using the Makefile:

```bash
make proto
```

### Adding New Services

1. Create a new directory: `api/proto/<service>/v1/`
2. Create your `.proto` file with proper package naming:

   ```proto
   syntax = "proto3";
   package narwhal.<service>.v1;
   option go_package = "github.com/narwhalmedia/narwhal/api/proto/<service>/v1;<service>pb";
   ```

3. Run `buf generate` to generate code
4. Implement the service interface in `internal/<service>/`

### Best Practices

1. **Versioning**: Always version your APIs (v1, v2, etc.)
2. **Package Naming**: Use `narwhal.<service>.v<version>`
3. **File Naming**: Use lowercase with underscores
4. **Message Naming**: Use PascalCase for messages and enums
5. **Field Naming**: Use lowercase with underscores
6. **Comments**: Document all services, methods, and non-obvious fields

### Common Types

Common types are defined in `common/v1/common.proto`:

- `MediaType`: Enum for media types (movie, series, music)
- `PaginationRequest/Response`: Standard pagination
- `SortOrder`: Standard sort ordering

### Style Guide

Follow the [Buf Style Guide](https://buf.build/docs/best-practices/style-guide) and Google's [Protocol Buffer Style Guide](https://developers.google.com/protocol-buffers/docs/style).

Key points:

- Use `optional` for nullable fields in proto3
- Prefer `google.protobuf.Timestamp` over int64 for timestamps
- Use `google.protobuf.Duration` for time intervals
- Include field masks for partial updates
- Use well-known types where appropriate
