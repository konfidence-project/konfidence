package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/konfidence-project/konfidence/internal/kden/log"
)

func Execute() {
	if err := execute(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func execute() (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Debug("panic recovered",
				"panic", r,
				"stack", string(debug.Stack()),
			)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	initCmd()
	return executeWith(ctx)
}
