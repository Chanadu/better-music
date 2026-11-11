package logger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"syscall"

	"github.com/lib/pq"
)

func ErrorDetails(err error) slog.Attr {
	var postgresError *pq.Error
	var networkError net.Error
	var systemError syscall.Errno

	switch {
	case errors.As(err, &postgresError):
		return slog.Group("error",
			"sqlstate", string(postgresError.Code),
			"condition", postgresError.Code.Name(),
			"constraint", postgresError.Constraint,
		)
	case errors.Is(err, context.Canceled):
		return slog.String("error", "context canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return slog.String("error", "context deadline exceeded")
	case errors.As(err, &systemError):
		return slog.String("error", systemError.Error())
	case errors.As(err, &networkError):
		return slog.Group("error", "type", fmt.Sprintf("%T", networkError), "timeout", networkError.Timeout())
	default:
		return slog.String("error_type", fmt.Sprintf("%T", err))
	}
}
