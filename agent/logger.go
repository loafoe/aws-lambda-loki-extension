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
	"github.com/grafana/loki-client-go/loki"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

var (
	labels = model.LabelSet{
		"job": "lambda",
	}
)

var logger = log.WithFields(log.Fields{"agent": "logsApiAgent"})

const (
	MaxPartSize = 5242880
)

// LokiLogger is the logger that writes the logs received from Logs API to Loki
type LokiLogger struct {
	client       *loki.Client
	functionName string
	endpoint     string
	key          string
	logLabels    *model.LabelSet
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
	cfg, err := loki.NewDefaultConfig(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error getting default config: %w", err)
	}
	// Auth
	username, _ := os.LookupEnv("LOKI_USERNAME")
	password, present := os.LookupEnv("LOKI_PASSWORD")
	if present {
		cfg.Client.BasicAuth = &config.BasicAuth{
			Username: username,
			Password: config.Secret(password),
		}
	}
	client, err := loki.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating loki client: %w", err)
	}
	lLabels := model.LabelSet{
		"job":         "lambda",
		"app":         model.LabelValue(fName),
		"function_name": model.LabelValue(fName),
	}

	// User provided labels in the format of where key/value separated by ";" and key and value seaprated by "="
	// example value "k1=v1; k2=v2; k3=v3"
	labels, present := os.LookupEnv("LOKI_LOG_LABELS")
	if present {
		entries := strings.Split(labels, ";")
		for _, e := range entries {
			parts := strings.Split(e, "=")
			labelName := strings.TrimSpace(parts[0])
			labelValue := strings.TrimSpace(parts[1])
			lLabels[model.LabelName(labelName)] = model.LabelValue(labelValue)

		}
	}

	return &LokiLogger{
		client:       client,
		functionName: fName,
		endpoint:     endpoint,
		key:          key,
		logLabels:    &lLabels,
		logBuffer:    buffer,
	}, nil
}

// PushLog writes the received logs to a buffer and takes actions depending on the current state of the logger.
func (l *LokiLogger) PushLog(log string) error {
	return l.client.Handle(*l.logLabels, time.Now(), log)
}

// Shutdown calls the function that should be executed before the program terminates
func (l *LokiLogger) Shutdown() error {
	return nil
}
