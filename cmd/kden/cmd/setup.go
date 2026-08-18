package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/konfidence-project/konfidence/internal/kden/log"
)

func Execute() {
	if err := execute(); err != nil {
		log.Error(err.Error())
	}
}

func execute() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	initCmd()
	return executeWith(ctx)
}
