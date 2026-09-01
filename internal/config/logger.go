package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

func LoggerInit() io.Closer {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = "ulas-service.log"
	}
	fmt.Println("logging to:", logFile)

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("failed to open log file (%s), falling back to stdout only: %v\n", logFile, err)
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, opts)))

		return io.NopCloser(nil)

	}

	multiWriter := io.MultiWriter(os.Stdout, file)

	handler := slog.NewJSONHandler(multiWriter, opts)

	slog.SetDefault(slog.New(handler))

	return file

}
