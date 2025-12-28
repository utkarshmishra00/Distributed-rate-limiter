# The name of the binary to produce
BINARY_NAME=rate-limiter

# Default target: print help
.DEFAULT_GOAL := help

.PHONY: help run redis-start redis-stop test test-refill clean

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

run: ## Start the Go server (ensure Redis is running first)
	go run main.go

redis-start: ## Start Redis in the background (Manual Method)
	redis-server --daemonize yes
	@echo "Redis started in background."

redis-stop: ## Stop the background Redis instance
	redis-cli shutdown
	@echo "Redis stopped."

test: ## Run a burst test (15 requests rapidly)
	@echo "Sending 15 requests... (Expect ~10 successes, ~5 failures)"
	@for i in {1..15}; do curl -s -w "\n" http://localhost:8080/ping; done

test-refill: ## Run a slow test (proof of refill logic)
	@echo "Sending 15 requests with 0.2s delay... (Expect ALL success)"
	@for i in {1..15}; do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/ping; sleep 0.2; done

clean: ## Remove build artifacts and Redis dump
	go clean
	rm -f dump.rdb
	@echo "Cleaned up."