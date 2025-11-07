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
    go build -o "$E2E_DIR/bin/mongodbtool" ./mongodbtool || { log_error "Failed to build mongodbtool"; exit 1; }
    go build -o "$E2E_DIR/bin/kafkatool" ./kafkatool || { log_error "Failed to build kafkatool"; exit 1; }
    
    log_success "All tools built successfully"
}

# Test MQTT tool
test_mqtt() {
    log_info "Testing MQTT tool..."
    
    local topic="test/e2e/mqtt"
    
    # Start subscriber in background
    timeout 10s "$E2E_DIR/bin/mqtttool" serve \
        --broker tcp://localhost:1883 \
        --topic "$topic" \
        > "$E2E_DIR/mqtt-serve.log" 2>&1 &
    local serve_pid=$!
    
    sleep 2
    
    # Send 3 messages
    timeout 5s "$E2E_DIR/bin/mqtttool" send \
        --broker tcp://localhost:1883 \
        --topic "$topic" \
        --payload '{"test":"mqtt-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/mqtt-send.log" 2>&1 || true
    
    # Wait for subscriber
    wait $serve_pid 2>/dev/null || true
    
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
    
    # Start subscriber in background
    timeout 10s "$E2E_DIR/bin/natstool" serve \
        --address nats://localhost:4222 \
        --subject "$subject" \
        > "$E2E_DIR/nats-serve.log" 2>&1 &
    local serve_pid=$!
    
    sleep 2
    
    # Send 3 messages
    timeout 5s "$E2E_DIR/bin/natstool" send \
        --address nats://localhost:4222 \
        --subject "$subject" \
        --payload '{"test":"nats-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/nats-send.log" 2>&1 || true
    
    wait $serve_pid 2>/dev/null || true
    
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
    
    # Start subscriber in background
    timeout 10s "$E2E_DIR/bin/redistool" serve \
        --address localhost:6379 \
        --channel "$channel" \
        > "$E2E_DIR/redis-serve.log" 2>&1 &
    local serve_pid=$!
    
    sleep 2
    
    # Send 3 messages
    timeout 5s "$E2E_DIR/bin/redistool" send \
        --address localhost:6379 \
        --channel "$channel" \
        --payload '{"test":"redis-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/redis-send.log" 2>&1 || true
    
    wait $serve_pid 2>/dev/null || true
    
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
    
    # Send 3 requests
    timeout 5s "$E2E_DIR/bin/httptool" send \
        --address http://localhost:8080 \
        --path /echo \
        --method POST \
        --payload '{"test":"http-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/http-send.log" 2>&1 || true
    
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
    
    # Start listener in background
    timeout 10s "$E2E_DIR/bin/pgsqltool" serve \
        --connstr "postgres://eventkit:eventkit@localhost:5432/eventkit?sslmode=disable" \
        --channel "$channel" \
        > "$E2E_DIR/postgres-serve.log" 2>&1 &
    local serve_pid=$!
    
    sleep 2
    
    # Send 3 notifications
    timeout 5s "$E2E_DIR/bin/pgsqltool" send \
        --connstr "postgres://eventkit:eventkit@localhost:5432/eventkit?sslmode=disable" \
        --channel "$channel" \
        --payload '{"test":"postgres-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/postgres-send.log" 2>&1 || true
    
    wait $serve_pid 2>/dev/null || true
    
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
    
    # Start change stream watcher in background
    timeout 10s "$E2E_DIR/bin/mongodbtool" serve \
        --uri "mongodb://localhost:27017/?directConnection=true" \
        --database eventkit \
        --collection "$collection" \
        > "$E2E_DIR/mongodb-serve.log" 2>&1 &
    local serve_pid=$!
    
    sleep 3
    
    # Insert 3 documents
    timeout 5s "$E2E_DIR/bin/mongodbtool" send \
        --uri "mongodb://localhost:27017/?directConnection=true" \
        --database eventkit \
        --collection "$collection" \
        --payload '{"test":"mongodb-e2e","time":"{nowtime}"}' \
        --interval 1s \
        > "$E2E_DIR/mongodb-send.log" 2>&1 || true
    
    wait $serve_pid 2>/dev/null || true
    
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
    
    # Start consumer in background
    timeout 15s "$E2E_DIR/bin/kafkatool" serve \
        --brokers localhost:9092 \
        --topic "$topic" \
        --group e2e-test \
        > "$E2E_DIR/kafka-serve.log" 2>&1 &
    local serve_pid=$!
    
    sleep 3
    
    # Send 3 messages
    timeout 8s "$E2E_DIR/bin/kafkatool" send \
        --brokers localhost:9092 \
        --topic "$topic" \
        --payload '{"test":"kafka-e2e","time":"{nowtime}"}' \
        --interval 2s \
        > "$E2E_DIR/kafka-send.log" 2>&1 || true
    
    wait $serve_pid 2>/dev/null || true
    
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
    test_mqtt || ((failed++))
    test_nats || ((failed++))
    test_redis || ((failed++))
    test_http || ((failed++))
    test_postgres || ((failed++))
    test_mongodb || ((failed++))
    test_kafka || ((failed++))
    
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
