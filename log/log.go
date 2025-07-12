package log

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sovamorco/errorx"
)

const (
	defaultFilePerm = 0o600
)

type zerologWriter struct {
	io.Writer

	errWriter io.Writer
}

func (z *zerologWriter) WriteLevel(level zerolog.Level, p []byte) (int, error) {
	writer := z.Writer
	if level == zerolog.ErrorLevel {
		writer = z.errWriter
	}

	n, err := writer.Write(p)
	if err != nil {
		return 0, errorx.Wrap(err, "write")
	}

	return n, nil
}

func InitLogger() zerolog.Logger {
	//nolint:reassign // that's the way of zerolog.
	zerolog.ErrorStackMarshaler = marshalErrorxStack

	writer := zerologWriter{
		Writer:    os.Stdout,
		errWriter: os.Stderr,
	}

	logger := zerolog.New(writer).With().Caller().Timestamp().Stack().Logger().Level(zerolog.DebugLevel)

	log.Logger = logger

	return logger
}

// if logs should be together - use same file name for both stdout and stderr.
func InitFileLogger(stdoutFileName, stderrFileName string) (zerolog.Logger, error) {
	//nolint:reassign // that's the way of zerolog.
	zerolog.ErrorStackMarshaler = marshalErrorxStack

	stdout, err := os.OpenFile(stdoutFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return zerolog.Logger{}, errorx.Wrap(err, "open stdout file")
	}

	stderr := stdout

	if stderrFileName != stdoutFileName {
		stderr, err = os.OpenFile(stderrFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, defaultFilePerm)
		if err != nil {
			return zerolog.Logger{}, errorx.Wrap(err, "open stderr file")
		}
	}

	writer := zerologWriter{
		Writer:    stdout,
		errWriter: stderr,
	}

	logger := zerolog.New(writer).With().Caller().Timestamp().Stack().Logger().Level(zerolog.DebugLevel)

	log.Logger = logger

	return logger, nil
}

func InitDevLogger() zerolog.Logger {
	logger := InitLogger()

	logger = logger.Level(zerolog.TraceLevel).Output(zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) {
			w.Out = os.Stderr
		},
	))
	log.Logger = logger

	return logger
}

//nolint:ireturn // that signature is required.
func marshalErrorxStack(ierr error) any {
	err := errorx.Cast(ierr)
	if err == nil {
		return nil
	}

	return err.MarshalStackTrace()
}
