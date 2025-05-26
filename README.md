# Narwhal

A cloud-native microservices media server built with Go, TypeScript, gRPC, and Kubernetes.

## Overview

Narwhal is a modern media server that provides:
- Media library management
- Automated media downloading
- HLS transcoding
- Streaming capabilities
- Web-based UI

## Architecture

The system is built using a microservices architecture with the following key components:

- Media Library Service: Manages media metadata and organization
- Download Service: Handles media file downloads
- Transcoding Service: Converts media to HLS format
- Streaming Service: Serves HLS content
- API Gateway: Exposes REST/gRPC endpoints
- Web UI: TypeScript/React frontend

## Prerequisites

- Go 1.21+
- Node.js 22+
- Docker
- Kubernetes cluster
- NATS JetStream
- PostgreSQL
- Redis

## Development Setup

1. Clone the repository:
```bash
git clone https://github.com/yourusername/narwhal.git
cd narwhal
```

2. Install Go dependencies:
```bash
go mod download
```

3. Generate protobuf code (includes fetching Google API definitions):
```bash
make proto
```

4. Install frontend dependencies:
```bash
cd web
npm install
```

5. Start development environment:
```bash
make dev
```

## Project Structure

```
narwhal/
├── cmd/                    # Service entry points
├── internal/              # Private application code
├── pkg/                   # Public libraries
├── api/                   # Protocol buffers and API definitions
├── web/                   # Frontend application
├── deploy/               # Kubernetes manifests
├── docs/                 # Documentation
└── tools/                # Development tools
```

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 