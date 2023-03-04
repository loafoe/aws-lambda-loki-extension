build:
	GOOS=linux GOARCH=amd64 go build -o bin/extensions/aws-lambda-loki-extension main.go

build-AwsLambdaLokiExtensionLayer:
	GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS_DIR)/extensions/aws-lambda-loki-extension main.go
	chmod +x $(ARTIFACTS_DIR)/extensions/aws-lambda-loki-extension

run-AwsLambdaLokiExtensionLayer:
	go run main.go
