package logger

import (
	"io"
	"log/slog"
	"os"
	"time"
)

func SetupLogger(fileEnabled bool, logFilePath string) error {
	var output io.Writer = os.Stdout
	if fileEnabled {
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		output = io.MultiWriter(os.Stdout, file)
	}

	logger := slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.DateTime))
			}
			return a
		},
	}))

	slog.SetDefault(logger)
	return nil
}
