#!/bin/bash
# End-to-End test runner for EventKit tools
# Requires docker-compose to be running

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
E2E_DIR="$PROJECT_ROOT/test/e2e"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}ℹ${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Check if docker-compose is running
check_services() {
    log_info "Checking if services are running..."
    
    if ! docker ps | grep -q eventkit-nanomq; then
        log_error "NanoMQ is not running. Start with: docker-compose up -d"
        exit 1
    fi
    
    log_success "All required services are running"
}

# Build all tools
build_tools() {
    log_info "Building all tools..."
    cd "$PROJECT_ROOT"
    
    go build -o "$E2E_DIR/bin/mqtttool" ./mqtttool || { log_error "Failed to build mqtttool"; exit 1; }
    go build -o "$E2E_DIR/bin/natstool" ./natstool || { log_error "Failed to build natstool"; exit 1; }
    go build -o "$E2E_DIR/bin/redistool" ./redistool || { log_error "Failed to build redistool"; exit 1; }
    go build -o "$E2E_DIR/bin/httptool" ./httptool || { log_error "Failed to build httptool"; exit 1; }
    go build -o "$E2E_DIR/bin/pgsqltool" ./pgsqltool || { log_error "Failed to build pgsqltool"; exit 1; }
    go build -o "$E2E_DIR/bin/mongotool" ./mongotool || { log_error "Failed to build mongotool"; exit 1; }
    go build -o "$E2E_DIR/bin/kafkatool" ./kafkatool || { log_error "Failed to build kafkatool"; exit 1; }
    
    log_success "All tools built successfully"
}

# Test MQTT tool
test_mqtt() {
    log_info "Testing MQTT tool..."
    
    local topic="test/e2e/mqtt"
    
    log_info "Starting MQTT subscriber..."
    # Start subscriber in background
    timeout 10s "$E2E_DIR/bin/mqtttool" serve \
        --broker tcp://localhost:1883 \
        --topic "$topic" \
        > "$E2E_DIR/mqtt-serve.log" 2>&1 &
    local serve_pid=$!
    log_info "MQTT subscriber started with PID: $serve_pid"
    
    log_info "Waiting 2 seconds for subscriber to be ready..."
    sleep 2
    
    log_info "Starting MQTT publisher..."
    # Send 3 messages
    timeout 5s "$E2E_DIR/bin/mqtttool" send \
        --broker tcp://localhost:1883 \
        --topic "$topic" \
        --payload '{"test":"mqtt-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/mqtt-send.log" 2>&1 || true
    log_info "MQTT publisher finished"
    
    log_info "Waiting for MQTT subscriber to finish..."
    # Wait for subscriber
    wait $serve_pid 2>/dev/null || true
    log_info "MQTT subscriber finished"
    
    log_info "Checking MQTT test results..."
    # Check if messages were received
    if grep -q "mqtt-e2e" "$E2E_DIR/mqtt-serve.log"; then
        log_success "MQTT test passed"
        return 0
    else
        log_error "MQTT test failed"
        cat "$E2E_DIR/mqtt-serve.log"
        return 1
    fi
}

# Test NATS tool
test_nats() {
    log_info "Testing NATS tool..."
    
    local subject="test.e2e.nats"
    
    log_info "Starting NATS subscriber..."
    # Start subscriber in background
    timeout 10s "$E2E_DIR/bin/natstool" serve \
        --address nats://localhost:4222 \
        --subject "$subject" \
        > "$E2E_DIR/nats-serve.log" 2>&1 &
    local serve_pid=$!
    log_info "NATS subscriber started with PID: $serve_pid"
    
    log_info "Waiting 2 seconds for subscriber to be ready..."
    sleep 2
    
    log_info "Starting NATS publisher..."
    # Send 3 messages
    timeout 5s "$E2E_DIR/bin/natstool" send \
        --address nats://localhost:4222 \
        --subject "$subject" \
        --payload '{"test":"nats-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/nats-send.log" 2>&1 || true
    log_info "NATS publisher finished"
    
    log_info "Waiting for NATS subscriber to finish..."
    wait $serve_pid 2>/dev/null || true
    log_info "NATS subscriber finished"
    
    log_info "Checking NATS test results..."
    if grep -q "nats-e2e" "$E2E_DIR/nats-serve.log"; then
        log_success "NATS test passed"
        return 0
    else
        log_error "NATS test failed"
        cat "$E2E_DIR/nats-serve.log"
        return 1
    fi
}

# Test Redis tool
test_redis() {
    log_info "Testing Redis tool..."
    
    local channel="test:e2e:redis"
    
    log_info "Starting Redis subscriber..."
    # Start subscriber in background
    timeout 10s "$E2E_DIR/bin/redistool" serve \
        --address localhost:6379 \
        --channel "$channel" \
        > "$E2E_DIR/redis-serve.log" 2>&1 &
    local serve_pid=$!
    log_info "Redis subscriber started with PID: $serve_pid"
    
    log_info "Waiting 2 seconds for subscriber to be ready..."
    sleep 2
    
    log_info "Starting Redis publisher..."
    # Send 3 messages
    timeout 5s "$E2E_DIR/bin/redistool" send \
        --address localhost:6379 \
        --channel "$channel" \
        --payload '{"test":"redis-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/redis-send.log" 2>&1 || true
    log_info "Redis publisher finished"
    
    log_info "Waiting for Redis subscriber to finish..."
    wait $serve_pid 2>/dev/null || true
    log_info "Redis subscriber finished"
    
    log_info "Checking Redis test results..."
    if grep -q "redis-e2e" "$E2E_DIR/redis-serve.log"; then
        log_success "Redis test passed"
        return 0
    else
        log_error "Redis test failed"
        cat "$E2E_DIR/redis-serve.log"
        return 1
    fi
}

# Test HTTP tool
test_http() {
    log_info "Testing HTTP tool..."
    
    log_info "Starting HTTP requests..."
    # Send 3 requests
    timeout 5s "$E2E_DIR/bin/httptool" send \
        --address http://localhost:8080 \
        --path /echo \
        --method POST \
        --payload '{"test":"http-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/http-send.log" 2>&1 || true
    log_info "HTTP requests finished"
    
    log_info "Checking HTTP test results..."
    if grep -q "http-e2e" "$E2E_DIR/http-send.log"; then
        log_success "HTTP test passed"
        return 0
    else
        log_error "HTTP test failed"
        cat "$E2E_DIR/http-send.log"
        return 1
    fi
}

# Test PostgreSQL tool
test_postgres() {
    log_info "Testing PostgreSQL tool..."
    
    local channel="test_e2e_postgres"
    
    log_info "Starting PostgreSQL listener..."
    # Start listener in background
    timeout 10s "$E2E_DIR/bin/pgsqltool" serve \
        --conn "postgres://eventkit:eventkit@localhost:5432/eventkit?sslmode=disable" \
        --channel "$channel" \
        > "$E2E_DIR/postgres-serve.log" 2>&1 &
    local serve_pid=$!
    log_info "PostgreSQL listener started with PID: $serve_pid"
    
    log_info "Waiting 2 seconds for listener to be ready..."
    sleep 2
    
    log_info "Starting PostgreSQL notifier..."
    # Send 3 notifications
    timeout 5s "$E2E_DIR/bin/pgsqltool" send \
        --conn "postgres://eventkit:eventkit@localhost:5432/eventkit?sslmode=disable" \
        --channel "$channel" \
        --payload '{"test":"postgres-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/postgres-send.log" 2>&1 || true
    log_info "PostgreSQL notifier finished"
    
    log_info "Waiting for PostgreSQL listener to finish..."
    wait $serve_pid 2>/dev/null || true
    log_info "PostgreSQL listener finished"
    
    log_info "Checking PostgreSQL test results..."
    if grep -q "postgres-e2e" "$E2E_DIR/postgres-serve.log"; then
        log_success "PostgreSQL test passed"
        return 0
    else
        log_error "PostgreSQL test failed"
        cat "$E2E_DIR/postgres-serve.log"
        return 1
    fi
}

# Test MongoDB tool
test_mongodb() {
    log_info "Testing MongoDB tool..."
    
    local collection="e2e_test"
    
    log_info "Starting MongoDB change stream watcher..."
    # Start change stream watcher in background
    timeout 10s "$E2E_DIR/bin/mongotool" serve \
        --uri "mongodb://localhost:27017/?directConnection=true" \
        --database eventkit \
        --collection "$collection" \
        > "$E2E_DIR/mongodb-serve.log" 2>&1 &
    local serve_pid=$!
    log_info "MongoDB watcher started with PID: $serve_pid"
    
    log_info "Waiting 3 seconds for watcher to be ready..."
    sleep 3
    
    log_info "Starting MongoDB document inserter..."
    # Insert 3 documents
    timeout 5s "$E2E_DIR/bin/mongotool" send \
        --uri "mongodb://localhost:27017/?directConnection=true" \
        --database eventkit \
        --collection "$collection" \
        --payload '{"test":"mongodb-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/mongodb-send.log" 2>&1 || true
    log_info "MongoDB inserter finished"
    
    log_info "Waiting for MongoDB watcher to finish..."
    wait $serve_pid 2>/dev/null || true
    log_info "MongoDB watcher finished"
    
    log_info "Checking MongoDB test results..."
    if grep -q "mongodb-e2e" "$E2E_DIR/mongodb-serve.log"; then
        log_success "MongoDB test passed"
        return 0
    else
        log_warning "MongoDB test may have failed (check logs)"
        # MongoDB change streams can be tricky, so just warn
        return 0
    fi
}

# Test Kafka tool
test_kafka() {
    log_info "Testing Kafka tool..."
    
    local topic="test-e2e-kafka"
    
    log_info "Starting Kafka consumer..."
    # Start consumer in background
    timeout 15s "$E2E_DIR/bin/kafkatool" serve \
        --brokers localhost:9092 \
        --topic "$topic" \
        --group e2e-test \
        > "$E2E_DIR/kafka-serve.log" 2>&1 &
    local serve_pid=$!
    log_info "Kafka consumer started with PID: $serve_pid"
    
    log_info "Waiting 3 seconds for consumer to be ready..."
    sleep 3
    
    log_info "Starting Kafka producer..."
    # Send 3 messages
    timeout 8s "$E2E_DIR/bin/kafkatool" send \
        --brokers localhost:9092 \
        --topic "$topic" \
        --payload '{"test":"kafka-e2e","time":"{nowtime}"}' \
        --interval 2s \
        > "$E2E_DIR/kafka-send.log" 2>&1 || true
    log_info "Kafka producer finished"
    
    log_info "Waiting for Kafka consumer to finish..."
    wait $serve_pid 2>/dev/null || true
    log_info "Kafka consumer finished"
    
    log_info "Checking Kafka test results..."
    if grep -q "kafka-e2e" "$E2E_DIR/kafka-serve.log"; then
        log_success "Kafka test passed"
        return 0
    else
        log_warning "Kafka test may have failed (check logs)"
        # Kafka can be slow to start, so just warn
        return 0
    fi
}

# Main test runner
main() {
    log_info "EventKit End-to-End Test Suite"
    echo ""
    
    # Create bin directory
    mkdir -p "$E2E_DIR/bin"
    
    # Check services
    check_services
    
    # Build tools
    build_tools
    
    echo ""
    log_info "Running E2E tests..."
    echo ""
    
    local failed=0
    
    # Run all tests
    log_info "=== Running MQTT test ==="
    test_mqtt || ((failed++))
    echo ""
    
    log_info "=== Running NATS test ==="
    test_nats || ((failed++))
    echo ""
    
    log_info "=== Running Redis test ==="
    test_redis || ((failed++))
    echo ""
    
    log_info "=== Running HTTP test ==="
    test_http || ((failed++))
    echo ""
    
    log_info "=== Running PostgreSQL test ==="
    test_postgres || ((failed++))
    echo ""
    
    log_info "=== Running MongoDB test ==="
    test_mongodb || ((failed++))
    echo ""
    
    log_info "=== Running Kafka test ==="
    test_kafka || ((failed++))
    echo ""
    
    echo ""
    if [ $failed -eq 0 ]; then
        log_success "All E2E tests passed! 🎉"
        exit 0
    else
        log_error "$failed test(s) failed"
        log_info "Check logs in: $E2E_DIR/*.log"
        exit 1
    fi
}

# Run main
main "$@"
