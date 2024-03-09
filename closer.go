package closer

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// Closer is a special object that exists because pressing Ctrl+C on a running go process will NOT EXECUTE ANY DEFERS.
// This is obviously very bad if we want to properly close databases and run cleanup on our routes.
//
// To do this, I made this struct to capture all cleanup functions that must be run with `AddCloseFn` and then
// upon calling `CloseGracefullyInCaseOfSigterm`, the closer will listen for the os SIGTERM signal. Before exiting, it will
// then heroically call all the registered close functions in FILO order just like a defer would.
type Closer struct {
	closerFunctions []func()
	messages        []string
	Logging         bool
}

// NewCloser creates a new Closer object
func NewCloser() *Closer {
	return &Closer{
		closerFunctions: []func(){},
		messages:        []string{},
		Logging:         true,
	}
}

// AddCloseFn adds a function to the closer that should be called for sure when the program is closed
func (c *Closer) AddCloseFn(message string, fn func()) {
	// because we want to execute the cleanup functions in FILO order, we must PREPEND instead of append
	// this is the idiomatic way of prepending in Go
	c.closerFunctions = append([]func(){fn}, c.closerFunctions...)
	c.messages = append([]string{message}, c.messages...)
}

// CloseAll calls all the registered close functions in FILO order
func (c *Closer) CloseAll() {
	for i, fn := range c.closerFunctions {
		if c.Logging {
			message := c.messages[i]
			fmt.Println(message)
		}
		fn()
	}
}

// CloseGracefullyInCaseOfSigterm listens for the os SIGTERM signal and calls CloseAll() when it is received
func (c *Closer) CloseGracefullyInCaseOfSigterm() {
	ch := make(chan os.Signal, 1) // Add buffer size of 1

	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-ch
		if c.Logging {
			fmt.Println("SIGTERM RECEIVED. CLOSING SERVER")
		}
		c.CloseAll()
		os.Exit(1)
	}()
}
