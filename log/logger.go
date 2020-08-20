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

var logger *logrus.Logger = nil

// Logger is a for now, a type alias of Logrus.FieldLogger that defines broad interface for logging.
type Logger = logrus.FieldLogger

// InitLogger sets the internal logger the given level and log file.
// This function should be called exactly once and subsequent calls return an error.
// Logs to stdout if logFile is an empty string.
func InitLogger(levelStr, logFile string) error {
	if logger != nil {
		return errors.New("logger already initialized")
	}

	newLogger := logrus.New()
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		return errors.WithStack(err)
	}
	newLogger.SetLevel(level)
	if logFile == "" {
		newLogger.SetOutput(os.Stdout)
	} else {
		f, err := os.OpenFile(filepath.Clean(logFile), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
		if err != nil {
			return errors.WithStack(err)
		}
		newLogger.SetOutput(f)
	}

	newLogger.SetFormatter(&customTextFormatter{logrus.TextFormatter{
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05 Z0700",
		DisableLevelTruncation: true,
	}})
	logger = newLogger
	return nil
}

// NewLoggerWithField returns a logger that logs with the given fields.
// It is derived from the internal logger instance of this package and uses the same log level and log file.
//
// If the internal logger instance is not initialized before this call, it is initialized to "debug" level
// and logs to the standard output (stdout).
func NewLoggerWithField(key string, value interface{}) Logger {
	if logger == nil {
		InitLogger("debug", "") // nolint: errcheck, gosec	// err will always be nil in this case.
	}
	l := logger.WithField(key, value)
	return l
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
