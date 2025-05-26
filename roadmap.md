# Narwhal Media Server - Implementation Roadmap

## Current Status

### ✅ Completed
- Core Domain Models (DDD patterns)
- Basic service structure
- Proto definitions for all services
- Development infrastructure (Docker Compose)
- Dependency injection framework setup
- Configuration management
- Structured logging with Zap
- Basic gRPC service wiring

### 🚧 In Progress
- Media service full implementation
- gRPC interceptors (logging done, others pending)

### ❌ Not Started
- Download service implementation
- Transcode service implementation
- Stream service implementation
- Event bus (NATS JetStream)
- Kubernetes components
- Service mesh integration

## Implementation Phases

### Phase 1: Core Service Functionality (Current)

#### 1.1 Dependency Injection & Service Wiring ✅
- [x] Configuration management via environment variables
- [x] Google Wire for dependency injection
- [x] Structured logging with Zap
- [x] Database connection pooling
- [x] Graceful shutdown handling

#### 1.2 Event Bus Integration ✅
- [x] NATS JetStream setup and client
- [x] Event publisher implementation
- [x] Event consumer framework
- [x] Saga orchestration for workflows
- [x] Dead letter queue handling

#### 1.3 Download Service ✅
- [x] HTTP download client with resume support
- [x] Torrent client integration
- [ ] Usenet client integration (placeholder)
- [x] Progress tracking and reporting
- [x] File validation (checksums)
- [x] Integration with event bus

#### 1.4 Transcode Service
- [ ] FFmpeg container and integration
- [ ] HLS multi-bitrate encoding
- [ ] Progress reporting
- [ ] Error handling and retry logic
- [ ] Output to storage (local/S3)

### Phase 2: Infrastructure & Operations

#### 2.1 Observability
- [x] Logging interceptor
- [ ] OpenTelemetry tracing
- [ ] Prometheus metrics
- [ ] Correlation ID propagation
- [ ] Distributed tracing with Jaeger

#### 2.2 Kubernetes Native
- [ ] Custom Resource Definitions (CRDs)
  - [ ] TranscodingJob CRD
  - [ ] DownloadJob CRD
  - [ ] MediaLibrary CRD
- [ ] Operators using Kubebuilder
- [ ] Helm charts
- [ ] Horizontal Pod Autoscaler configs

#### 2.3 Service Mesh (Istio)
- [ ] mTLS between services
- [ ] Traffic management policies
- [ ] Circuit breakers
- [ ] Retry policies
- [ ] Canary deployment support

### Phase 3: Production Readiness

#### 3.1 Storage & Content Delivery
- [ ] MinIO integration for S3-compatible storage
- [ ] CDN integration patterns
- [ ] Caching strategies with Redis
- [ ] Static file serving optimization

#### 3.2 Security
- [ ] JWT authentication
- [ ] API Gateway with auth
- [ ] RBAC for services
- [ ] Input validation
- [ ] Rate limiting

#### 3.3 Testing & Quality
- [ ] Unit test coverage >80%
- [ ] Integration tests with testcontainers
- [ ] E2E tests with Kind
- [ ] Load testing with k6
- [ ] Chaos engineering tests

## Quick Start for Development

1. **Start infrastructure:**
   ```bash
   make dev
   ```

2. **Build services:**
   ```bash
   make build
   ```

3. **Run tests:**
   ```bash
   make test
   ```

4. **Run a specific service:**
   ```bash
   # Set environment variables
   export $(cat .env.example | xargs)
   
   # Run media service
   ./bin/media
   ```

## Next Immediate Tasks

1. **Complete Media Service gRPC Methods** - Implement all proto-defined methods
2. **NATS Event Bus** - Replace placeholder event publisher
3. **Download Service** - Implement actual download functionality
4. **Integration Tests** - Add testcontainers-based tests

## Architecture Decisions

See `/docs/adr/` for Architecture Decision Records (when created).

## Contributing

When implementing new features:
1. Follow DDD principles
2. Write tests first (TDD)
3. Add proper logging and metrics
4. Update this roadmap
5. Document major decisions in ADRs