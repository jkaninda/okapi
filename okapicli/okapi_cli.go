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
	"reflect"
	"strconv"
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
	o       *okapi.Okapi
	flagSet *pflag.FlagSet
	flags   map[string]interface{}
	// structPtr holds a pointer to the struct being populated from CLI flags, used for writing back parsed values after resolution
	structPtr interface{}
	// envMappings maps CLI flag names to environment variable names for easy lookup during env var application
	envMappings map[string]string
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
		o:           o,
		flagSet:     pflag.NewFlagSet(appName, pflag.ExitOnError),
		flags:       make(map[string]interface{}),
		envMappings: make(map[string]string),
	}
}

// Default creates a CLI manager with a default Okapi instance
func Default() *CLI {
	return New(okapi.Default())
}

// Okapi returns the underlying Okapi instance
func (c *CLI) Okapi() *okapi.Okapi {
	return c.o
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

// Duration adds a time.Duration flag with optional shorthand
func (c *CLI) Duration(name, shorthand string, duration time.Duration, usage string) *CLI {
	c.flagSet.DurationP(name, shorthand, duration, usage)
	return c
}

// ParseFlags parses the command line flags
func (c *CLI) ParseFlags() error {
	// First apply environment variables to override defaults
	if err := c.applyEnvVars(); err != nil {
		return err
	}
	// Parse command-line arguments
	if err := c.flagSet.Parse(os.Args[1:]); err != nil {
		return err
	}

	// Populate struct with final values (after env + CLI resolution)
	if c.structPtr != nil {
		if err := c.populateStruct(); err != nil {
			return err
		}
	}

	return nil
}

// Parse is an alias for ParseFlags
func (c *CLI) Parse() error {
	return c.ParseFlags()
}

func (c *CLI) MustParse() *CLI {
	if err := c.Parse(); err != nil {
		panic(fmt.Errorf("cli parse failed: %w", err))
	}
	return c
}

// populateStruct writes final flag values back into the struct
func (c *CLI) populateStruct() error {
	val := reflect.ValueOf(c.structPtr).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !field.IsExported() {
			continue
		}

		cliName := strings.TrimSpace(field.Tag.Get("cli"))
		if cliName == "" {
			continue
		}

		// Write parsed value back to struct
		switch field.Type.Kind() {
		case reflect.String:
			if v, err := c.flagSet.GetString(cliName); err == nil {
				fieldVal.SetString(v)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if v, err := c.flagSet.GetInt(cliName); err == nil {
				fieldVal.SetInt(int64(v))
			}
		case reflect.Bool:
			if v, err := c.flagSet.GetBool(cliName); err == nil {
				fieldVal.SetBool(v)
			}
		case reflect.Float32, reflect.Float64:
			if v, err := c.flagSet.GetFloat64(cliName); err == nil {
				fieldVal.SetFloat(v)
			}
		}
	}

	return nil
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

// GetDuration retrieves a time.Duration flag value
func (c *CLI) GetDuration(name string) time.Duration {
	val, _ := c.flagSet.GetDuration(name)
	return val
}

// FromStruct registers CLI flags from struct tags.
// Supported tags:
//   - cli:     flag name (required to register flag)
//   - short:   shorthand letter (optional)
//   - desc:    description text (optional)
//   - env:     environment variable name to read from (optional)
//   - default: default value (optional; otherwise uses field's current value)
//
// Supported types: string, int*, bool, float*
func (c *CLI) FromStruct(v interface{}) *CLI {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		panic("FromStruct requires a non-nil pointer to a struct")
	}

	c.structPtr = v
	typ := val.Elem().Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Elem().Field(i)

		if !field.IsExported() {
			continue // Skip unexported fields
		}

		cliName := strings.TrimSpace(field.Tag.Get("cli"))
		if cliName == "" {
			continue // Skip fields without a cli tag
		}

		shorthand := strings.TrimSpace(field.Tag.Get("short"))
		description := strings.TrimSpace(field.Tag.Get("desc"))
		envVar := strings.TrimSpace(field.Tag.Get("env"))
		defaultTag := strings.TrimSpace(field.Tag.Get("default"))

		// Register flag + capture env mapping
		switch field.Type.Kind() {
		case reflect.String:
			defValue := defaultTag
			if defValue == "" && fieldVal.Kind() == reflect.String {
				defValue = fieldVal.String()
			}
			c.flagSet.StringP(cliName, shorthand, defValue, description)
			if envVar != "" {
				c.envMappings[cliName] = envVar
			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// Check if it's time.Duration
			if field.Type == reflect.TypeOf(time.Duration(0)) {
				defValue := time.Duration(0)
				if defaultTag != "" {
					if v, err := time.ParseDuration(defaultTag); err == nil {
						defValue = v
					}
				} else if fieldVal.Type() == reflect.TypeOf(time.Duration(0)) {
					defValue = fieldVal.Interface().(time.Duration)
				}
				c.flagSet.DurationP(cliName, shorthand, defValue, description)
				if envVar != "" {
					c.envMappings[cliName] = envVar
				}
			} else {
				defValue := 0
				if defaultTag != "" {
					if v, err := strconv.Atoi(defaultTag); err == nil {
						defValue = v
					}
				} else if fieldVal.Kind() == reflect.Int {
					defValue = int(fieldVal.Int())
				}
				c.flagSet.IntP(cliName, shorthand, defValue, description)
				if envVar != "" {
					c.envMappings[cliName] = envVar
				}
			}

		case reflect.Bool:
			defValue := false
			if defaultTag != "" {
				if v, err := strconv.ParseBool(defaultTag); err == nil {
					defValue = v
				}
			} else if fieldVal.Kind() == reflect.Bool {
				defValue = fieldVal.Bool()
			}
			c.flagSet.BoolP(cliName, shorthand, defValue, description)
			if envVar != "" {
				c.envMappings[cliName] = envVar
			}

		case reflect.Float32, reflect.Float64:
			defValue := 0.0
			if defaultTag != "" {
				if v, err := strconv.ParseFloat(defaultTag, 64); err == nil {
					defValue = v
				}
			} else if fieldVal.Kind() == reflect.Float64 {
				defValue = fieldVal.Float()
			}
			c.flagSet.Float64P(cliName, shorthand, defValue, description)
			if envVar != "" {
				c.envMappings[cliName] = envVar
			}

		default:
			// Skip unsupported types
			continue
		}
	}

	return c
}

// WithConfig registers CLI flags from struct tags.
// Supported tags:
//   - cli:     flag name (required to register flag)
//   - short:   shorthand letter (optional)
//   - desc:    description text (optional)
//   - env:     environment variable name to read from (optional)
//   - default: default value (optional; otherwise uses field's current value)
//
// Supported types: string, int*, bool, float*
func (c *CLI) WithConfig(cfg interface{}) *CLI {
	c.structPtr = cfg
	c.FromStruct(cfg)
	return c
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
		if err := c.o.Start(); err != nil {
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
		if err := c.o.StopWithContext(ctx); err != nil {
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

// applyEnvVars reads environment variables and sets corresponding flags
func (c *CLI) applyEnvVars() error {
	for flagName, envVar := range c.envMappings {
		if envValue := os.Getenv(envVar); envValue != "" {
			if err := c.flagSet.Set(flagName, envValue); err != nil {
				return fmt.Errorf("failed to set flag %q from env %s=%q: %w",
					flagName, envVar, envValue, err)
			}
		}
	}
	return nil
}

// LoadConfig loads configuration from a JSON or YAML file into a struct.
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
