build-up:
	@docker compose -p cartapi up --build

test:
	@go test ./...