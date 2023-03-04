// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

package agent

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

var logger = log.WithFields(log.Fields{"agent": "logsApiAgent"})

const (
	MaxPartSize = 5242880
)

// LokiLogger is the logger that writes the logs received from Logs API to Loki
type LokiLogger struct {
	functionName string
	endpoint     string
	key          string
	logBuffer    *bytes.Buffer
}

// NewLokiLogger returns an S3 Logger
func NewLokiLogger() (*LokiLogger, error) {
	fName := strings.ToLower(os.Getenv("AWS_LAMBDA_FUNCTION_NAME"))
	endpoint, present := os.LookupEnv("LOKI_PUSH_ENDPOINT")
	if !present {
		return nil, errors.New("environment variable LOKI_PUSH_ENDPOINT is not set")
	} else {
		fmt.Println("Sending logs to:", endpoint)
	}
	ts := int(time.Now().UnixNano() / 1000000)
	timestampMilli := strconv.Itoa(ts)
	key := fmt.Sprintf("%s-%s-%s.log", fName, timestampMilli, uuid.New())
	buffer := bytes.NewBuffer([]byte(""))
	buffer.Grow(2 * MaxPartSize)

	return &LokiLogger{
		functionName: fName,
		endpoint:     endpoint,
		key:          key,
		logBuffer:    buffer,
	}, nil
}

// PushLog writes the received logs to a buffer and takes actions depending on the current state of the logger.
func (l *LokiLogger) PushLog(log string) error {
	l.logBuffer.Write([]byte(log))
	// Should send logs here to Loki
	return nil
}

// Shutdown calls the function that should be executed before the program terminates
func (l *LokiLogger) Shutdown() error {
	return nil
}
