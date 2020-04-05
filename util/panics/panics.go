package panics

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/kaspanet/kaspad/logs"
)

// HandlePanic recovers panics, log them, runs an optional panicHandler,
// and then initiates a clean shutdown.
func HandlePanic(log *logs.Logger, goroutineStackTrace []byte) {
	err := recover()
	if err == nil {
		return
	}

	panicHandlerDone := make(chan struct{})
	go func() {
		log.Criticalf("Fatal error: %+v", err)
		if goroutineStackTrace != nil {
			log.Criticalf("Goroutine stack trace: %s", goroutineStackTrace)
		}
		log.Criticalf("Stack trace: %s", debug.Stack())
		log.Backend().Close()
		close(panicHandlerDone)
	}()

	const panicHandlerTimeout = 5 * time.Second
	select {
	case <-time.After(panicHandlerTimeout):
		fmt.Fprintln(os.Stderr, "Couldn't handle a fatal error. Exiting...")
	case <-panicHandlerDone:
	}
	log.Criticalf("Exiting")
	os.Exit(1)
}

// GoroutineWrapperFunc returns a goroutine wrapper function that handles panics and writes them to the log.
func GoroutineWrapperFunc(log *logs.Logger) func(func()) {
	return func(f func()) {
		stackTrace := debug.Stack()
		go func() {
			defer HandlePanic(log, stackTrace)
			f()
		}()
	}
}

// AfterFuncWrapperFunc returns a time.AfterFunc wrapper function that handles panics.
func AfterFuncWrapperFunc(log *logs.Logger) func(d time.Duration, f func()) *time.Timer {
	return func(d time.Duration, f func()) *time.Timer {
		stackTrace := debug.Stack()
		return time.AfterFunc(d, func() {
			defer HandlePanic(log, stackTrace)
			f()
		})
	}
}

// Exit writes the given exit reason to the given log, waits for
// it to finish, and exits.
func Exit(log *logs.Logger, reason string) {
	exitHandlerDone := make(chan struct{})
	go func() {
		log.Criticalf("Exiting: %s", reason)
		log.Backend().Close()
		close(exitHandlerDone)
	}()

	const exitHandlerTimeout = 5 * time.Second
	select {
	case <-time.After(exitHandlerTimeout):
		fmt.Fprintln(os.Stderr, "Couldn't exit gracefully.")
	case <-exitHandlerDone:
	}
	os.Exit(1)
}
