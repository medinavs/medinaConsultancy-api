build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go

deploy: build
	serverless deploy

clean:
	rm -f bootstrap

# API server locally on :8080
dev:
	@echo "Starting local dev server on http://localhost:8080..."
	@set -a && [ -f .env ] && . .env; set +a && \
		HANDLER_MODE=local go run main.go

# billing process locally
billing-local:
	@echo "Running billing locally..."
	@set -a && [ -f .env ] && . .env; set +a && \
		HANDLER_MODE=billing-local go run main.go

# invoke billing Lambda on AWS
invoke-billing:
	serverless invoke -f billing

# invoke API health check on AWS
invoke-health:
	serverless invoke -f api --data '{"requestContext":{"http":{"method":"GET","path":"/health"}},"rawPath":"/health"}'