# Builds the Docker base image for loki log extension to sends the Lambda
# function logs to configured LOKI_PUSH_ENDPOINT
FROM golang:1.20 AS BuildStage
WORKDIR /lokiextension
COPY . .
RUN GOOS=linux GOARCH=amd64 go build
RUN chmod +x bin/extensions/aws-lambda-loki-extension

# Copy from the Build stage 
# Note: This base image is to copy the extension layer to your application base image - it can't be  run directly 
FROM scratch 
WORKDIR /opt/extensions
COPY --from=BuildStage /lokiextension/bin/extensions .
