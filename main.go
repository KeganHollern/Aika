package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func newInterruptContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// probably not the right way to say "whenever anything happens on this channel"
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func main() {
	ctx, cancel := newInterruptContext(context.Background())
	defer cancel()

	// do things that exit w/ ctx cancellation

}
