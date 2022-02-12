package main

import (
	"context"
	"fmt"
	"math/rand"
	"os/signal"
	"syscall"
	"time"

	"github.com/spy16/connote/cli"
)

var (
	Commit    = "N/A"
	Version   = "N/A"
	BuildTime = "N/A"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	rand.Seed(time.Now().UnixNano())

	versionStr := fmt.Sprintf("Version %s (commit %s built on %s)", Version, Commit, BuildTime)
	cli.Execute(ctx, versionStr)
}
