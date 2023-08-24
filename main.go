// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"

	"github.com/golang-collections/go-datastructures/queue"
	"github.com/loafoe/aws-lambda-loki-extension/agent"
	"github.com/loafoe/aws-lambda-loki-extension/extension"
	"github.com/loafoe/aws-lambda-loki-extension/logsapi"
	log "github.com/sirupsen/logrus"
)

// InitialQueueSize is the initial size set for the synchronous logQueue
const InitialQueueSize = 5

func main() {
	extensionName := path.Base(os.Args[0])
	printPrefix := fmt.Sprintf("[%s]", extensionName)
	logger := log.WithFields(log.Fields{"agent": extensionName})

	enableLokiLoggingExtension := enableLokiExtension()

	extensionClient := extension.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"))

	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-sigs
		cancel()
		logger.Info(printPrefix, "Received", s)
		logger.Info(printPrefix, "Exiting")
	}()

	// Register extension as soon as possible
	_, err := extensionClient.Register(ctx, extensionName)
	if err != nil {
		panic(err)
	}

	var logsApiAgent *agent.HttpAgent = nil
	var flushLogQueue func(force bool) = nil

	// send subscription request to Lambda service to receive logging telemetry
	// that can be forwarded to grafana loki
	if enableLokiLoggingExtension {
		// Create Loki Logger
		logsApiLogger, err := agent.NewLokiLogger()
		if err != nil {
			logger.Fatal(err)
		}

		// A synchronous queue that is used to put logs from the goroutine (producer)
		// and process the logs from main goroutine (consumer)
		logQueue := queue.New(InitialQueueSize)
		// Helper function to empty the log queue
		var logsStr string = ""
		flushLogQueue = func(force bool) {
			for !(logQueue.Empty() && (force || strings.Contains(logsStr, string(logsapi.RuntimeDone)))) {
				logs, err := logQueue.Get(1)
				if err != nil {
					logger.Error(printPrefix, err)
					return
				}
				logsStr = fmt.Sprintf("%v", logs[0])
				err = logsApiLogger.PushLog(logsStr)
				if err != nil {
					logger.Error(printPrefix, err)
					return
				}
			}
		}
		// Create Logs API agent
		logsApiAgent, err = agent.NewHttpAgent(logsApiLogger, logQueue)
		if err != nil {
			logger.Fatal(err)
		}

		// Subscribe to logs API
		// Logs start being delivered only after the subscription happens.
		agentID := extensionClient.ExtensionID
		err = logsApiAgent.Init(agentID)
		if err != nil {
			logger.Fatal(err)
		}
	}

	// Will block until invoke or shutdown event is received or cancelled via the context.
	for {
		select {
		case <-ctx.Done():
			return
		default:
			logger.Info(printPrefix, " Waiting for event...")
			// This is a blocking call
			res, err := extensionClient.NextEvent(ctx)
			if err != nil {
				logger.Info(printPrefix, "Error:", err)
				logger.Info(printPrefix, "Exiting")
				return
			}
			// Flush log queue in here after waking up
			if enableLokiLoggingExtension {
				flushLogQueue(false)
			}
			// Exit if we receive a SHUTDOWN event
			if res.EventType == extension.Shutdown {
				logger.Info(printPrefix, "Received SHUTDOWN event")

				if enableLokiLoggingExtension {
					flushLogQueue(true)
					logsApiAgent.Shutdown()
				}

				logger.Info(printPrefix, "Exiting")
				return
			}
		}
	}

}

// function by default returns true, unless 'LOKI_EXTENSION_ENABLED' enironmenent variable value is set to 'false'
func enableLokiExtension() bool {
	enableExtension := true
	value, present := os.LookupEnv("LOKI_EXTENSION_ENABLED")
	if present {
		enableExtension, _ = strconv.ParseBool(value)
	}
	return enableExtension
}
