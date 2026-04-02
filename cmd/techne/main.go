package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Edcko/techne-code/cmd/techne/cli"
)

// version is set at build time via ldflags.
// Example: go build -ldflags "-X main.version=1.0.0" ./cmd/techne/
var version = "dev"

func main() {
	// Handle signals for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel context on signal
	go func() {
		<-sigs
		cancel()
	}()

	if err := cli.Execute(ctx, version); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
