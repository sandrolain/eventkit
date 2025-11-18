# EventKit

![EventKit](./eventkit.png)

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/sandrolain/eventkit)](https://goreportcard.com/report/github.com/sandrolain/eventkit)

EventKit is a comprehensive collection of command-line tools designed for testing and interacting with various protocols and event brokers. It provides unified interfaces for CoAP, MQTT, NATS, Kafka, HTTP, Redis, Google Pub/Sub, PostgreSQL, MongoDB, and Git, making it ideal for testing event-driven systems, debugging message flows, and performance evaluation.

## Features

✅ **Multi-Protocol Support** - Works with 10 different protocols and event brokers  
✅ **Advanced Template System** - Dynamic payload generation with 15+ placeholders, custom variables, file includes, and wrappers  
✅ **MIME Type Support** - Handles text/plain, application/json, and application/cbor with auto-detection  
✅ **Deterministic Testing** - Reproducible test data with configurable seed  
✅ **Secure File Handling** - Optional file includes with sandboxing and caching  
✅ **Graceful Shutdown** - Proper signal handling (SIGINT/SIGTERM) for clean exits  
✅ **Periodic Messaging** - Send messages at configurable intervals  
✅ **Performance Testing** - Built-in timing and throughput measurement  
✅ **Flexible CLI** - Consistent command-line interface across all tools  

## Installation

Install all tools:

```bash
go install github.com/sandrolain/eventkit/...@latest
```

Or install individual tools:

```bash
go install github.com/sandrolain/eventkit/coaptool@latest
go install github.com/sandrolain/eventkit/mqtttool@latest
go install github.com/sandrolain/eventkit/natstool@latest
go install github.com/sandrolain/eventkit/kafkatool@latest
go install github.com/sandrolain/eventkit/httptool@latest
go install github.com/sandrolain/eventkit/redistool@latest
go install github.com/sandrolain/eventkit/pubsubtool@latest
go install github.com/sandrolain/eventkit/pgsqltool@latest
go install github.com/sandrolain/eventkit/mongotool@latest
go install github.com/sandrolain/eventkit/gittool@latest
```

## Available Tools

### 🔵 CoAP Tool

Test CoAP (Constrained Application Protocol) servers - ideal for IoT and constrained environments.

```bash
# Send POST requests periodically
coaptool send --server coap://localhost:5683 --path /sensors --payload '{"temp": {{rand}}}' --interval 5s

# Receive CoAP requests
coaptool receive --address :5683 --path /sensors
```

**Key Options:**

- `--server` - CoAP server URL (e.g., coap://host:port)
- `--path` - Resource path

### 📡 MQTT Tool

Test MQTT brokers with publish/subscribe messaging.

```bash
# Publish messages to topic
mqtttool send --server tcp://localhost:1883 --topic sensors/temperature --payload '{"value": {{rand}}}' --interval 5s

# Subscribe to topic
mqtttool receive --server tcp://localhost:1883 --topic sensors/#
```

**Key Options:**

- `--server` - MQTT broker URL (tcp://host:port)
- `--topic` - MQTT topic (supports wildcards in receive: +, #)
- `--qos` - Quality of Service (0, 1, 2)

### ⚡ NATS Tool

Test NATS messaging systems with subject-based routing.

```bash
# Publish to subject
natstool send --server nats://localhost:4222 --topic events.user.created --payload '{"id": "{{uuid}}"}' --interval 5s

# Subscribe to subject pattern
natstool receive --server nats://localhost:4222 --topic events.user.*
```

**Key Options:**

- `--server` - NATS server URL (nats://host:port)
- `--topic` - NATS subject (supports wildcards: *, >)

### 📨 Kafka Tool

Test Apache Kafka with producer/consumer operations.

```bash
# Produce messages
kafkatool send --server localhost:9092 --topic user-events --payload '{"action": "login", "time": "{{nowtime}}"}' --interval 10s

# Consume messages from topic
kafkatool receive --server localhost:9092 --topic user-events --group my-consumer-group
```

**Key Options:**

- `--server` - Kafka broker address (host:port)
- `--topic` - Kafka topic name
- `--group` - Consumer group ID (for receive)
- `--partition` - Specific partition (optional)

### 🌐 HTTP Tool

Test HTTP endpoints with POST requests or run simple HTTP servers.

```bash
# Send POST requests
httptool send --dest http://localhost:8080/api/events --payload '{"type": "test", "ts": "{{nowtime}}"}' --interval 5s

# Start HTTP server to receive requests
httptool receive --address :8080 --path /api/events
```

**Key Options:**

- `--dest` - Destination URL (for send)
- `--address` - Listen address (for receive)
- `--path` - HTTP path
- `--method` - HTTP method (default: POST)

### 🔴 Redis Tool

Test Redis Pub/Sub messaging.

```bash
# Publish to channel
redistool send --server localhost:6379 --topic notifications --payload '{"msg": "Hello", "time": "{{nowtime}}"}' --interval 5s

# Subscribe to channel
redistool receive --server localhost:6379 --topic notifications
```

**Key Options:**

- `--server` - Redis server address (host:port)
- `--topic` - Redis channel name
- `--password` - Redis password (optional)

### ☁️ Google Pub/Sub Tool

Test Google Cloud Pub/Sub messaging.

```bash
# Publish messages
pubsubtool send --project my-gcp-project --topic events-topic --payload '{"event": "click", "user": "{{uuid}}"}' --interval 5s

# Subscribe to messages
pubsubtool receive --project my-gcp-project --subscription events-sub
```

**Key Options:**

- `--project` - GCP project ID
- `--topic` - Pub/Sub topic name (for send)
- `--subscription` - Subscription name (for receive)

### 🐘 PostgreSQL Tool

Test PostgreSQL LISTEN/NOTIFY for async notifications.

```bash
# Send notifications
pgsqltool send --conn "postgres://user:pass@localhost/mydb" --channel app-events --payload '{"type": "update"}' --interval 10s

# Listen for notifications
pgsqltool receive --conn "postgres://user:pass@localhost/mydb" --channel app-events
```

**Key Options:**

- `--conn` - PostgreSQL connection string
- `--channel` - NOTIFY channel name

### 🍃 MongoDB Tool

Test MongoDB with document insertion and Change Streams monitoring.

```bash
# Insert documents periodically
mongotool send --server mongodb://localhost:27017 --database testdb --collection events \
  --payload '{"type": "sensor", "value": {{rand}}, "timestamp": "{{nowtime}}"}' --interval 5s

# Watch Change Streams for real-time updates
mongotool serve --server mongodb://localhost:27017 --database testdb --collection events
```

**Key Options:**

- `--server` - MongoDB connection string (mongodb://host:port)
- `--database` - Database name
- `--collection` - Collection name
- `--username` - MongoDB username (optional)
- `--password` - MongoDB password (optional)

**Note:** Change Streams require a MongoDB replica set. The tool automatically adds an `_insertedAt` timestamp to each document.

### 📦 Git Tool

Automated Git commits for testing CI/CD pipelines or Git hooks.

```bash
# Periodic commits to repository
gittool send \
  --remote https://github.com/user/test-repo.git \
  --branch main \
  --filename data.log \
  --payload "Automated update at {{nowtime}}" \
  --message "Auto-commit from eventkit" \
  --interval 30s
```

**Key Options:**

- `--remote` - Git repository URL
- `--branch` - Target branch (default: main)
- `--filename` - File to modify
- `--message` - Commit message
- `--username`, `--password` - Authentication (for HTTPS)

## Payload Interpolation

All tools support dynamic payload generation with a powerful template system.

### Standard Placeholders

Default delimiters are `{{` and `}}`.

| Placeholder | Description | Example Output |
|-------------|-------------|----------------|
| `{{json}}` | Random JSON object | `{"key1":"val","key2":123}` |
| `{{cbor}}` | Random CBOR data | Binary CBOR-encoded data |
| `{{nowtime}}` | Current timestamp (RFC3339) | `2024-01-15T14:30:00Z` |
| `{{datetime}}` | Alias for `{{nowtime}}` | `2024-01-15T14:30:00Z` |
| `{{rand}}` | Random integer | `42857291` |
| `{{uuid}}` | UUID v4 | `550e8400-e29b-41d4-a716-446655440000` |
| `{{counter}}` | Incrementing counter (process-local) | `1`, `2`, `3`, ... |
| `{{sentence}}` | Random sentence | `The quick brown fox jumps` |
| `{{sentiment}}` | Random sentiment text | `positive`, `negative`, `neutral` |

### Template Variables

Inject custom values with `--template-var`:

```bash
mqtttool send --server tcp://localhost:1883 --topic alerts \
  --template-var env=production \
  --template-var region=us-east \
  --payload '{"env": "{{var:env}}", "region": "{{var:region}}", "ts": "{{nowtime}}"}'
```

### File Includes

Include file contents with `{{file:path}}` (requires `--allow-file-reads`):

```bash
# Include file content (disabled by default for security)
httptool send --dest http://localhost:8080/upload \
  --allow-file-reads \
  --file-root /safe/data \
  --payload '{{file:config.json}}'

# Enable caching for repeated file reads
httptool send --dest http://localhost:8080/upload \
  --allow-file-reads \
  --cache-files \
  --payload '{{file:template.json}}' \
  --interval 1s
```

**Security Notes:**

- File reads are **disabled by default**
- Use `--allow-file-reads` to enable
- Use `--file-root` to restrict access to a specific directory subtree
- Use `--cache-files` to cache file content (process-lifetime cache)

### Wrappers

Control how placeholders are inserted:

| Wrapper | Description | Example |
|---------|-------------|----------|
| `{{raw:placeholder}}` | Insert raw bytes | `{{raw:json}}` → `{"a":1}` |
| `{{str:placeholder}}` | Insert JSON-quoted string | `{{str:json}}` → `"{\"a\":1}"` |

```bash
# Insert JSON as quoted string
mqtttool send --server tcp://localhost:1883 --topic data \
  --payload '{"metadata": {{str:json}}, "raw_data": {{raw:cbor}}}'
```

### Custom Delimiters

Change placeholder delimiters with `--template-open` and `--template-close`:

```bash
mqtttool send --server tcp://localhost:1883 --topic data \
  --template-open '<%' --template-close '%>' \
  --payload '{"time": "<%nowtime%>", "id": "<%uuid%>"}'
```

### Deterministic Testing

Use `--seed` for reproducible random data:

```bash
# Same seed produces same random values
mqtttool send --server tcp://localhost:1883 --topic test \
  --seed 12345 \
  --payload '{"id": "{{uuid}}", "value": {{rand}}}'
```

### Basic Example

```bash
mqtttool send \
  --server tcp://localhost:1883 \
  --topic sensors/data \
  --payload '{"id": "{{uuid}}", "timestamp": "{{nowtime}}", "value": {{rand}}}' \
  --mime application/json \
  --interval 5s
```

## Common Options

All send commands support these options:

### Message Options

- `--interval` - Time between messages (e.g., `10s`, `1m`, `5m30s`, `1h`)
- `--payload` - Message content (supports template interpolation)
- `--mime` - MIME type (`text/plain`, `application/json`, `application/cbor`); auto-detected if empty
- `--size` - Payload size for auto-generated content (in bytes)

### Template Options

- `--template-open` - Opening delimiter for placeholders (default: `{{`)
- `--template-close` - Closing delimiter for placeholders (default: `}}`)
- `--template-var key=value` - Define custom template variable (repeatable)
- `--seed N` - Deterministic seed for random data generation
- `--allow-file-reads` - Enable `{{file:path}}` placeholders (disabled by default)
- `--file-root path` - Restrict file reads to directory subtree
- `--cache-files` - Enable caching for `{{file:path}}` includes

### Connection Aliases

Flag aliases for server/destination (all tools accept both):

- `--server` / `--address` / `--broker` / `--conn` (deprecated)
- `--dest` / `--destination` / `--remote` (deprecated)

## Use Cases

### IoT Testing

```bash
# Simulate temperature sensor
coaptool send --server coap://iot.example.com:5683 --path /sensors/temp \
  --payload '{"device": "sensor-01", "temp": {{rand}}, "time": "{{nowtime}}"}' --interval 30s
```

### Microservices Events

```bash
# Publish user events to Kafka
kafkatool send --server kafka.local:9092 --topic user.events \
  --payload '{"user_id": "{{uuid}}", "action": "login", "ts": {{counter}}}' --interval 2s
```

### CI/CD Pipeline Testing

```bash
# Trigger Git commits for testing webhooks
gittool send --remote https://github.com/org/test.git --branch develop \
  --filename ci-test.txt --payload "Test run {{uuid}}" --interval 1m
```

### Database Async Notifications

```bash
# PostgreSQL event notification
pgsqltool send --conn "postgres://app:secret@db:5432/events" \
  --channel cache-invalidate --payload '{"table": "users", "id": {{rand}}}'
```

### Real-Time Data Monitoring

```bash
# Monitor MongoDB Change Streams
mongotool serve --server mongodb://localhost:27017 --database myapp --collection logs

# Generate test data
mongotool send --server mongodb://localhost:27017 --database myapp --collection logs \
  --payload '{"level": "info", "msg": "Test event {{uuid}}", "time": "{{nowtime}}"}' --interval 3s
```

### Advanced Template Usage

```bash
# Deterministic testing with seeded random data
kafkatool send --server localhost:9092 --topic test-events \
  --seed 42 \
  --payload '{"id": "{{uuid}}", "value": {{rand}}}' \
  --interval 1s

# Using template variables for environment-specific data
httptool send --dest http://api.example.com/events \
  --template-var env=staging \
  --template-var version=1.2.3 \
  --payload '{"env": "{{var:env}}", "version": "{{var:version}}", "ts": "{{nowtime}}"}'

# File includes with sandboxing and caching
mqtttool send --server tcp://localhost:1883 --topic config-updates \
  --allow-file-reads \
  --file-root /app/configs \
  --cache-files \
  --payload '{{file:service-config.json}}' \
  --interval 10s

# Using wrappers to embed JSON in strings
natstool send --server nats://localhost:4222 --topic events.data \
  --payload '{"metadata": {{str:json}}, "counter": {{counter}}}'
```

## Development

### Requirements

- Go 1.25 or higher
- Protocol-specific services (MQTT broker, NATS server, etc.) for testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run linter
golangci-lint run ./...
```

### Project Structure

```text
eventkit/
├── pkg/
│   ├── common/         # Shared utilities (signal handling, CLI helpers)
│   ├── testpayload/    # Payload generation and interpolation
│   └── toolutil/       # Common tool functions (formatting, flags)
├── coaptool/           # CoAP tool
├── mqtttool/           # MQTT tool
├── natstool/           # NATS tool
├── kafkatool/          # Kafka tool
├── httptool/           # HTTP tool
├── redistool/          # Redis tool
├── pubsubtool/         # Google Pub/Sub tool
├── pgsqltool/          # PostgreSQL tool
├── mongotool/          # MongoDB tool
└── gittool/            # Git tool
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass and coverage is ≥80%
5. Run linter: `golangci-lint run ./...`
6. Commit using conventional commits format (`feat:`, `fix:`, etc.)
7. Push to the branch
8. Open a Pull Request

For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with:

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Paho MQTT](https://github.com/eclipse/paho.mqtt.golang) - MQTT client
- [NATS.go](https://github.com/nats-io/nats.go) - NATS client
- [kafka-go](https://github.com/segmentio/kafka-go) - Kafka client
- [go-redis](https://github.com/redis/go-redis) - Redis client
- [go-git](https://github.com/go-git/go-git) - Git operations
- [fasthttp](https://github.com/valyala/fasthttp) - High-performance HTTP

---

**EventKit** - Making event-driven system testing simple and efficient.
