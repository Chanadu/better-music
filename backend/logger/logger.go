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
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				attr.Value = slog.StringValue(attr.Value.Time().Format(time.DateTime))
			}
			return attr
		},
	}))

	slog.SetDefault(logger)
	return nil
}
