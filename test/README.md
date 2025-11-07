# EventKit Testing Guide

This directory contains integration and end-to-end tests for EventKit tools.

## Test Structure

```
test/
├── integration/          # Integration tests with testcontainers
│   └── containers_test.go
├── e2e/                 # End-to-end tests
│   ├── run-e2e.sh       # E2E test runner
│   └── bin/             # Built tools (generated)
└── coap-server/         # CoAP test server
    ├── Dockerfile
    └── server.py
```

## Prerequisites

- Docker and Docker Compose
- Go 1.25+
- [Task](https://taskfile.dev/) (task runner)

## Quick Start

### 1. Start Test Services

Start all required services using Docker Compose:

```bash
# Using the helper script
./scripts/manage-services.sh start

# Or using Task
task services-start

# Or directly with docker compose
docker compose up -d
```

This starts:

- NanoMQ (MQTT broker) - port 1883
- NATS with JetStream - port 4222
- Redis - port 6379
- PostgreSQL - port 5432
- MongoDB (with replica set) - port 27017
- Kafka (KRaft mode) - port 9092
- HTTP echo server - port 8080
- CoAP server - port 5683 (UDP)

### 2. Check Service Health

```bash
./scripts/manage-services.sh health
# or
task services-health
```

### 3. Run Tests

#### Unit Tests

```bash
go test ./pkg/... -cover
# or
task test
```

#### Integration Tests (with testcontainers)

```bash
go test ./test/integration/... -v -timeout 10m
# or
task test-integration
```

#### End-to-End Tests

```bash
./test/e2e/run-e2e.sh
# or
task test-e2e
```

#### All Tests

```bash
task test-all
```

## Integration Tests

Integration tests use [testcontainers-go](https://golang.testcontainers.org/) to spin up real service containers for testing.

Each test:

1. Starts a containerized service
2. Waits for it to be ready
3. Verifies connectivity
4. Cleans up the container

Run with:

```bash
go test ./test/integration/... -v
```

Skip integration tests:

```bash
go test ./test/integration/... -short
```

## End-to-End Tests

E2E tests verify complete workflows by:

1. Building all CLI tools
2. Running send/serve commands against real services
3. Verifying data flow between components

Each test:

- Starts a "serve" process (subscriber/listener/watcher)
- Runs a "send" process (publisher/sender)
- Verifies messages were received
- Captures logs for debugging

### E2E Test Coverage

- ✅ MQTT (NanoMQ)
- ✅ NATS (with and without JetStream)
- ✅ Redis (Pub/Sub)
- ✅ HTTP (request/response)
- ✅ PostgreSQL (LISTEN/NOTIFY)
- ✅ MongoDB (insert + change streams)
- ✅ Kafka (producer/consumer)

### Viewing E2E Logs

All E2E test logs are saved in `test/e2e/`:

```bash
ls -la test/e2e/*.log
```

Example logs:

- `mqtt-send.log` - MQTT publisher output
- `mqtt-serve.log` - MQTT subscriber output
- `nats-send.log` - NATS publisher output
- etc.

## Service Management

### Start Services

```bash
./scripts/manage-services.sh start
```

### Stop Services

```bash
./scripts/manage-services.sh stop
```

### View Logs

```bash
# All services
./scripts/manage-services.sh logs

# Specific service
./scripts/manage-services.sh logs nanomq
./scripts/manage-services.sh logs mongodb
```

### Service Status

```bash
./scripts/manage-services.sh status
```

### Clean Volumes (removes all data)

```bash
./scripts/manage-services.sh clean
```

## Service URLs

When services are running, they're available at:

| Service | URL | Notes |
|---------|-----|-------|
| NanoMQ | `tcp://localhost:1883` | MQTT broker |
| NATS | `nats://localhost:4222` | With JetStream enabled |
| Redis | `localhost:6379` | Pub/Sub + Streams |
| PostgreSQL | `postgres://eventkit:eventkit@localhost:5432/eventkit` | LISTEN/NOTIFY |
| MongoDB | `mongodb://localhost:27017` | Replica set for change streams |
| Kafka | `localhost:9092` | KRaft mode (no Zookeeper) |
| HTTP Server | `http://localhost:8080` | Echo server |
| CoAP Server | `coap://localhost:5683` | UDP + TCP |

## Troubleshooting

### Services won't start

```bash
# Check Docker is running
docker ps

# Check ports aren't already in use
lsof -i :1883  # MQTT
lsof -i :4222  # NATS
lsof -i :6379  # Redis
lsof -i :5432  # PostgreSQL
lsof -i :27017 # MongoDB
lsof -i :9092  # Kafka
```

### MongoDB replica set issues

```bash
# Re-initialize replica set
docker exec eventkit-mongodb mongosh --eval 'rs.initiate({_id: "rs0", members: [{_id: 0, host: "localhost:27017"}]})'

# Check replica set status
docker exec eventkit-mongodb mongosh --eval 'rs.status()'
```

### Kafka is slow to start

Kafka can take 30-60 seconds to fully start. Wait a bit longer or check logs:

```bash
docker logs eventkit-kafka
```

### E2E tests fail

1. Ensure all services are running: `task services-health`
2. Check service logs: `docker compose logs`
3. Review E2E test logs: `ls test/e2e/*.log`
4. Rebuild tools: `task build`

### Integration tests timeout

Increase timeout:

```bash
go test ./test/integration/... -timeout 15m
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      
      - uses: arduino/setup-task@v2
        with:
          version: 3.x
      
      # Unit tests (no Docker needed)
      - name: Unit Tests
        run: task test
      
      # Integration tests (testcontainers handles Docker)
      - name: Integration Tests
        run: task test-integration
      
      # E2E tests (requires services)
      - name: Start Services
        run: task services-start
      
      - name: E2E Tests
        run: task test-e2e
      
      - name: Stop Services
        if: always()
        run: task services-stop
```

## Development Workflow

1. **Start services once**

   ```bash
   task dev-setup
   ```

2. **Develop and test**

   ```bash
   # Edit code
   vim mqtttool/send.go
   
   # Run tests
   task test
   
   # Run E2E for specific tool
   ./test/e2e/run-e2e.sh  # will run all, or edit script
   ```

3. **Clean up when done**

   ```bash
   task services-stop
   ```

## Adding New Tests

### Integration Test

1. Add test to `test/integration/containers_test.go`
2. Use testcontainers pattern:

   ```go
   func TestNewTool(t *testing.T) {
       ctx := context.Background()
       req := testcontainers.ContainerRequest{...}
       container, err := testcontainers.GenericContainer(ctx, ...)
       defer container.Terminate(ctx)
       // ... test logic
   }
   ```

### E2E Test

1. Add test function to `test/e2e/run-e2e.sh`:

   ```bash
   test_newtool() {
       log_info "Testing new tool..."
       # Start serve process
       # Run send process
       # Verify output
   }
   ```

2. Call it from `main()` function
3. Update service list in `check_services()`

## Best Practices

1. **Always check service health before testing**
2. **Use timeouts** to prevent hanging tests
3. **Capture logs** for debugging failed tests
4. **Clean up resources** in test cleanup/defer
5. **Use realistic payloads** with testpayload placeholders
6. **Run integration tests in isolation** (they manage their own containers)
7. **Keep E2E tests simple** - test happy path, not edge cases

## Resources

- [Testcontainers Go](https://golang.testcontainers.org/)
- [Docker Compose](https://docs.docker.com/compose/)
- [NanoMQ](https://nanomq.io/)
- [NATS](https://docs.nats.io/)
