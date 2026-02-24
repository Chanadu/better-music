package logger

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/Chanadu/better-music/config"
)

func SetupLogger(config *config.Config) {
	file, _ := os.OpenFile(config.Logs.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	multiWriter := io.MultiWriter(os.Stdout, file)

	logger := slog.New(slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
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
}
