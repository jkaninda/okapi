/*
 *  MIT License
 *
 * Copyright (c) 2026 Jonas Kaninda
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in all
 *  copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 *  SOFTWARE.
 */

package okapicli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jkaninda/okapi"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	// SIGINT is the interrupt signal
	SIGINT = syscall.SIGINT
	// SIGTERM is the termination signal
	SIGTERM = syscall.SIGTERM
)

type CLI struct {
	app     *okapi.Okapi
	flagSet *pflag.FlagSet
	flags   map[string]interface{}
}

// RunOptions configures the Run behavior
type RunOptions struct {
	// ShutdownTimeout is the maximum time to wait for graceful shutdown
	ShutdownTimeout time.Duration

	// Signals are the OS signals to listen for (defaults to SIGINT, SIGTERM)
	Signals []os.Signal

	// OnStart is called right before the server starts
	OnStart func()

	// OnStarted is called after the server starts successfully
	OnStarted func()

	// OnShutdown is called before shutdown begins
	OnShutdown func()
}

// New creates a new CLI manager for the Okapi
// You can optionally provide a custom application name
func New(o *okapi.Okapi, name ...string) *CLI {
	appName := "Okapi-CLI"
	if len(name) > 0 {
		appName = name[0]
	}
	return &CLI{
		app:     o,
		flagSet: pflag.NewFlagSet(appName, pflag.ExitOnError),
		flags:   make(map[string]interface{}),
	}
}

// String adds a string flag with optional shorthand
func (c *CLI) String(name, shorthand, defaultValue, usage string) *CLI {
	c.flagSet.StringP(name, shorthand, defaultValue, usage)
	return c
}

// Int adds an integer flag with optional shorthand
func (c *CLI) Int(name, shorthand string, defaultValue int, usage string) *CLI {
	c.flagSet.IntP(name, shorthand, defaultValue, usage)
	return c
}

// Bool adds a boolean flag with optional shorthand
func (c *CLI) Bool(name, shorthand string, defaultValue bool, usage string) *CLI {
	c.flagSet.BoolP(name, shorthand, defaultValue, usage)
	return c
}

// Float adds a float64 flag with optional shorthand
func (c *CLI) Float(name, shorthand string, defaultValue float64, usage string) *CLI {
	c.flagSet.Float64P(name, shorthand, defaultValue, usage)
	return c
}

// ParseFlags parses the command line flags
func (c *CLI) ParseFlags() error {
	return c.flagSet.Parse(os.Args[1:])
}

// Get retrieves a flag value by name
func (c *CLI) Get(name string) interface{} {
	if val, ok := c.flags[name]; ok {
		switch v := val.(type) {
		case *string:
			return *v
		case *int:
			return *v
		case *bool:
			return *v
		case *float64:
			return *v
		}
	}
	return nil
}

// GetString retrieves a string flag value
func (c *CLI) GetString(name string) string {
	val, _ := c.flagSet.GetString(name)
	return val
}

// GetInt retrieves an int flag value
func (c *CLI) GetInt(name string) int {
	val, _ := c.flagSet.GetInt(name)
	return val
}

// GetBool retrieves a bool flag value
func (c *CLI) GetBool(name string) bool {
	val, _ := c.flagSet.GetBool(name)
	return val
}

// GetFloat retrieves a float64 flag value
func (c *CLI) GetFloat(name string) float64 {
	val, _ := c.flagSet.GetFloat64(name)
	return val
}

// DefaultRunOptions returns default run options
func defaultRunOptions() *RunOptions {
	return &RunOptions{
		ShutdownTimeout: 30 * time.Second,
		Signals:         []os.Signal{SIGINT, SIGTERM},
	}
}

// RunServer starts Okapi and waits for shutdown signals.
// It handles graceful shutdown automatically
func (c *CLI) RunServer(opts ...*RunOptions) error {
	options := defaultRunOptions()
	if len(opts) > 0 && opts[0] != nil {
		options = opts[0]
	}

	// Set default signals if none provided
	if len(options.Signals) == 0 {
		options.Signals = []os.Signal{SIGINT, SIGTERM}
	}
	// Call OnStart callback if provided
	if options.OnStart != nil {
		options.OnStart()
	}

	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		if err := c.app.Start(); err != nil {
			serverErrors <- err
		}
	}()

	// Call OnStarted callback if provided
	if options.OnStarted != nil {
		go func() {
			time.Sleep(100 * time.Millisecond)
			options.OnStarted()
		}()
	}

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, options.Signals...)

	// Block until receiving a signal or an error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-quit:
		// Call OnShutdown callback if provided
		if options.OnShutdown != nil {
			options.OnShutdown()
		}

		// Create a context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), options.ShutdownTimeout)
		defer cancel()

		// Attempt a graceful shutdown
		if err := c.app.StopWithContext(ctx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
	}

	return nil
}

// Run starts Okapi using default options and waits for shutdown signals.
// It handles graceful shutdown automatically.
//
// It is a shortcut for RunServer(nil)
func (c *CLI) Run() error {
	return c.RunServer(nil)
}

// LoadConfig loads configuration from a file (JSON or YAML) into a struct
func (c *CLI) LoadConfig(path string, v interface{}) error {
	if path == "" {
		return fmt.Errorf("config path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		if err = json.Unmarshal(data, v); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err = yaml.Unmarshal(data, v); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s (supported: .json, .yaml, .yml)", ext)
	}

	return nil
}
