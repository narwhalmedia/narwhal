# Narwhal Media Platform

A unified media platform that combines the functionality of Sonarr, Radarr, and Plex into a microservices architecture built with Go.

## Architecture

Narwhal is designed as a cloud-native microservices platform with the following core services:

- **Media Library Service**: Manages media files, metadata, and library organization
- **Content Acquisition Service**: Handles searching indexers and managing downloads
- **Streaming Service**: Provides HLS/DASH adaptive bitrate streaming
- **Transcoding Service**: Handles video transcoding with hardware acceleration
- **User Service**: Manages authentication, authorization, and user profiles
- **Analytics Service**: Tracks viewing history and provides recommendations

## Technology Stack

- **Language**: Go
- **Communication**: gRPC (internal), REST/GraphQL (external)
- **Streaming**: HLS/DASH with FFmpeg
- **Container**: Docker & Kubernetes
- **Database**: PostgreSQL (per-service databases)
- **Caching**: Redis + in-memory
- **Message Bus**: NATS/RabbitMQ for event-driven architecture

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker & Docker Compose
- Protocol Buffers compiler (protoc)

### Building

```bash
# Clone the repository
git clone https://github.com/narwhalmedia/narwhal.git
cd narwhal

# Install dependencies
go mod download

# Build all services
make build

# Or build a specific service
go build -o bin/library ./cmd/library
```

### Running

```bash
# Run with Docker Compose
docker-compose up

# Or run individual services
./bin/library
```

### Development

```bash
# Run tests
go test ./...

# Run with hot reload
air -c .air.toml

# Generate protobuf files
make generate
```

## Project Structure

```
narwhal/
├── cmd/                    # Service entry points
│   ├── library/
│   ├── acquisition/
│   └── streaming/
├── internal/              # Private service implementations
│   └── library/
│       ├── domain/       # Business logic
│       ├── repository/   # Data access
│       ├── service/      # Service layer
│       └── handler/      # gRPC/HTTP handlers
├── pkg/                   # Shared packages
│   ├── models/           # Domain models
│   ├── interfaces/       # Common interfaces
│   ├── events/           # Event definitions
│   └── utils/            # Utilities
├── api/                   # API definitions
│   ├── proto/            # gRPC protobuf files
│   └── openapi/          # REST API specs
└── deployments/          # Deployment configs
    ├── docker/
    └── kubernetes/
```

## Contributing

This project is currently in the planning and initial implementation phase. Contributions are welcome!

## License

[License information to be added]