package log

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sovamorco/errorx"
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
		return 0, errorx.Decorate(err, "write")
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
