#!/bin/bash
# Helper script to manage EventKit test infrastructure

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${GREEN}ℹ${NC} $1"; }
log_warn() { echo -e "${YELLOW}⚠${NC} $1"; }
log_error() { echo -e "${RED}✗${NC} $1"; }
log_success() { echo -e "${GREEN}✓${NC} $1"; }

# Start all services
start_services() {
    log_info "Starting EventKit services..."
    cd "$PROJECT_ROOT"
    
    # Make init-mongo.sh executable
    chmod +x scripts/init-mongo.sh
    
    docker-compose up -d
    
    log_info "Waiting for services to be ready..."
    sleep 10
    
    # Initialize MongoDB replica set
    log_info "Initializing MongoDB replica set..."
    docker exec eventkit-mongodb mongosh --eval 'rs.initiate({_id: "rs0", members: [{_id: 0, host: "localhost:27017"}]})' 2>/dev/null || log_warn "MongoDB replica set may already be initialized"
    
    log_success "All services started"
    show_status
}

# Stop all services
stop_services() {
    log_info "Stopping EventKit services..."
    cd "$PROJECT_ROOT"
    docker-compose down
    log_success "All services stopped"
}

# Show service status
show_status() {
    log_info "Service Status:"
    echo ""
    docker-compose ps
}

# Show service logs
show_logs() {
    cd "$PROJECT_ROOT"
    if [ -n "$1" ]; then
        docker-compose logs -f "$1"
    else
        docker-compose logs -f
    fi
}

# Clean volumes
clean_volumes() {
    log_warn "This will remove all data from services"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Stopping services and removing volumes..."
        cd "$PROJECT_ROOT"
        docker-compose down -v
        log_success "Volumes removed"
    else
        log_info "Cancelled"
    fi
}

# Run integration tests
run_integration_tests() {
    log_info "Running integration tests..."
    cd "$PROJECT_ROOT"
    go test -v ./test/integration/... -timeout 5m
}

# Run E2E tests
run_e2e_tests() {
    log_info "Running E2E tests..."
    cd "$PROJECT_ROOT"
    ./test/e2e/run-e2e.sh
}

# Health check
health_check() {
    log_info "Checking service health..."
    
    local failed=0
    
    # NanoMQ
    if docker ps | grep -q eventkit-nanomq; then
        log_success "NanoMQ is running"
    else
        log_error "NanoMQ is not running"
        ((failed++))
    fi
    
    # NATS
    if docker ps | grep -q eventkit-nats; then
        log_success "NATS is running"
    else
        log_error "NATS is not running"
        ((failed++))
    fi
    
    # Redis
    if docker ps | grep -q eventkit-redis; then
        log_success "Redis is running"
    else
        log_error "Redis is not running"
        ((failed++))
    fi
    
    # PostgreSQL
    if docker ps | grep -q eventkit-postgres; then
        log_success "PostgreSQL is running"
    else
        log_error "PostgreSQL is not running"
        ((failed++))
    fi
    
    # MongoDB
    if docker ps | grep -q eventkit-mongodb; then
        log_success "MongoDB is running"
    else
        log_error "MongoDB is not running"
        ((failed++))
    fi
    
    # Kafka
    if docker ps | grep -q eventkit-kafka; then
        log_success "Kafka is running"
    else
        log_error "Kafka is not running"
        ((failed++))
    fi
    
    # HTTP Server
    if docker ps | grep -q eventkit-httpserver; then
        log_success "HTTP Server is running"
    else
        log_error "HTTP Server is not running"
        ((failed++))
    fi
    
    echo ""
    if [ $failed -eq 0 ]; then
        log_success "All services are healthy"
        return 0
    else
        log_error "$failed service(s) are not running"
        return 1
    fi
}

# Show help
show_help() {
    cat << EOF
EventKit Infrastructure Manager

Usage: $0 [command]

Commands:
    start       Start all services
    stop        Stop all services
    restart     Restart all services
    status      Show service status
    logs        Show logs (optional: service name)
    health      Run health check
    clean       Stop services and remove volumes
    test-int    Run integration tests
    test-e2e    Run E2E tests
    help        Show this help

Examples:
    $0 start                 # Start all services
    $0 logs nanomq          # Show NanoMQ logs
    $0 test-e2e             # Run E2E tests

EOF
}

# Main
case "${1:-help}" in
    start)
        start_services
        ;;
    stop)
        stop_services
        ;;
    restart)
        stop_services
        start_services
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs "$2"
        ;;
    health)
        health_check
        ;;
    clean)
        clean_volumes
        ;;
    test-int)
        run_integration_tests
        ;;
    test-e2e)
        run_e2e_tests
        ;;
    help|*)
        show_help
        ;;
esac
