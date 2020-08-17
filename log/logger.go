// Copyright (c) 2020 - for information on the respective copyright owner
// see the NOTICE file and/or the repository at
// https://github.com/hyperledger-labs/perun-node
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Logger is a for now, a type alias of Logrus.Logger.
type Logger = logrus.Logger

// NewLogger returns a logger set to the given level and log file.
// Supported log levels are "error" and "info".
// Logs to stdout if logFile is an empty string.
func NewLogger(levelStr, logFile string) (*Logger, error) {
	logger := logrus.New()

	if levelStr != "info" && levelStr != "error" {
		return nil, errors.New("Unsupported log level, use info or error")
	}
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	logger.SetLevel(level)

	if logFile == "" {
		logger.SetOutput(os.Stdout)
	} else {
		f, err := os.OpenFile(filepath.Clean(logFile), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		logger.SetOutput(f)
	}

	logger.SetFormatter(&customTextFormatter{logrus.TextFormatter{
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05 Z0700",
		DisableLevelTruncation: true,
	}})
	return logger, nil

}

// customTextFormatter is defined to override default formating options for log entry.
type customTextFormatter struct {
	logrus.TextFormatter
}

// Format modifies the default logging format.
func (f *customTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	originalText, err := f.TextFormatter.Format(entry)
	return append([]byte("▶ "), originalText...), err
}
