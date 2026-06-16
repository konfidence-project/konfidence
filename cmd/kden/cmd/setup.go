package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/konfidence-project/konfidence/internal/kden/log"
)

func Execute() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	initCmd()
	if err := executeWith(ctx); err != nil {
		log.Errorf("failed to start kden CLI: %v", err)
		os.Exit(1)
	}
}
