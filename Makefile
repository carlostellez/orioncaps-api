.PHONY: run test cover lint build-lambda

run:
	go run ./cmd/server

test:
	go test ./... -race

cover:
	go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out

lint:
	go vet ./...

# Artefacto para AWS Lambda (runtime provided.al2023, arm64): dist/lambda.zip con `bootstrap` en la raiz
build-lambda:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o dist/bootstrap ./cmd/lambda
	cd dist && zip -j lambda.zip bootstrap
