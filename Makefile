run:
	docker-compose up

test:
	go test -timeout 30s -v ./...
