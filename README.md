# aws-lambda-loki-extension

Lambda extension to push function logs directly to a Loki cluster

This extension: 
* Subscribes to receive platform and function logs
* Runs with a main and a helper goroutine: The main goroutine registers to ExtensionAPI and process its invoke and shutdown events (see nextEvent call). The helper goroutine:
    - starts a local HTTP server at the provided port (default 1234) that receives requests from Logs API
    - pushes the logs to a Loki handler
* Loki clients pushes the logs in batches to the cluster

## Compile package and dependencies

To run this extension, you will need to ensure that your build architecture matches that of the Lambda execution environment by compiling with `GOOS=linux` and `GOARCH=amd64` if you are not running in a Linux environment.

Building and saving package into a `bin/extensions` directory:
```bash
$ cd aws-lambda-loki-extension
$ GOOS=linux GOARCH=amd64 go build -o bin/extensions/aws-lambda-loki-extension main.go
$ chmod +x bin/extensions/aws-lambda-loki-extension
```

## Layer Setup Process
The extensions .zip file should contain a root directory called `extensions/`, where the extension executables are located. In this sample project we must include the `aws-lambda-loki-extension` binary.

Creating zip package for the extension:
```bash
$ cd bin
$ zip -r extension.zip extensions/
```

Publish a new layer using the `extension.zip` and capture the produced layer arn in `layer_arn`. If you don't have jq command installed, you can run only the aws cli part and manually pass the layer arn to `aws lambda update-function-configuration`.
```bash
layer_arn=$(aws lambda publish-layer-version --layer-name "aws-lambda-loki-extension" --region "<use your region>" --zip-file  "fileb://extension.zip" | jq -r '.LayerVersionArn')
```

Add the newly created layer version to a Lambda function.
```bash
aws lambda update-function-configuration --region <use your region> --function-name <your function name> --layers $layer_arn
```

## Function Invocation and Extension Execution
> Note: You need to add `LOKI_PUSH_ENDPOINT` environment variable to your lambda function. This value is used to configure the Loki client.

> Note: If the Loki endpoint is password protected you'll need to set `LOKI_USERNAME` and `LOKI_PASSWORD` as well.

After invoking the function and receiving the shutdown event, you should now see log messages from the extension written to the Loki cluster.
