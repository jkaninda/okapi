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
	"fmt"
	"log/slog"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/okapitest"
)

type serverConfig struct {
	Port    int           `cli:"port"   short:"p" desc:"HTTP server port"        env:"APP_PORT"   default:"8080"`
	Host    string        `cli:"host"   short:"h" desc:"Server hostname"        env:"APP_HOST"   default:"localhost"`
	Debug   bool          `cli:"debug"  short:"d" desc:"Enable debug mode"      env:"APP_DEBUG" default:"false"`
	Config  string        `cli:"config" short:"c" desc:"Path to config file"   default:"config.yaml"`
	Timeout time.Duration `cli:"timeout" short:"t" desc:"Request timeout" default:"30s"`
}

func setOSArgs(args ...string) func() {
	oldArgs := os.Args
	os.Args = append([]string{os.Args[0]}, args...)
	return func() { os.Args = oldArgs }
}

func TestNew(t *testing.T) {
	app := okapi.New()

	// Set up CLI flags
	cli := New(app, "Okapi").
		String("config", "c", "", "Path to provider configuration file").
		Int("port", "p", 8000, "HTTP server port").
		Bool("debug", "d", false, "Enable debug mode").
		Float("size", "s", -1e-1000, "Size of the server")

	err := cli.ParseFlags()
	if err != nil {
		t.Error(err)
	}
	size := cli.GetFloat("size")
	if size != -1e-1000 {
		t.Error("Expected size -1e-1000, got", size)
	}
	restore := setOSArgs("--port", "7000", "--debug", "true", "--config", "config.yaml")
	defer restore()
	port := cli.GetInt("port")
	fmt.Println("Port Flag:", port)

}
func TestRun(t *testing.T) {
	app := okapi.New()

	// Set up CLI flags
	cli := New(app, "Okapi Test").
		String("config", "c", "", "Path to provider configuration file").
		Int("port", "p", 8000, "HTTP server port").
		Bool("debug", "d", false, "Enable debug mode")

	err := cli.Parse()
	if err != nil {
		t.Error(err)
	}
	restore := setOSArgs("--port", "7000", "--debug", "true", "-c", "config.yaml")
	defer restore()

	port := cli.GetInt("port")
	app.WithPort(port)
	fmt.Println("Port Flag:", port)
	okapitest.GracefulExitAfter(5 * time.Second)
	//
	if err = cli.Run(); err != nil {
		t.Fatal("Server error", "error", err)
	}
}
func TestCLI_RunServer(t *testing.T) {
	app := okapi.New()

	// Set up CLI flags
	cli := New(app, "Okapi Test").
		String("config", "c", "", "Path to provider configuration file").
		Int("port", "p", 8000, "HTTP server port").
		Bool("debug", "d", false, "Enable debug mode").
		Duration("timeout", "t", 30*time.Second, "Request timeout")

	err := cli.ParseFlags()
	if err != nil {
		t.Error(err)
	}
	port := cli.GetInt("port")
	app.WithPort(port)
	fmt.Println("Port Flag:", port)
	okapitest.GracefulExitAfter(5 * time.Second)
	//

	if err = cli.RunServer(&RunOptions{
		ShutdownTimeout: 30 * time.Second,
		Signals:         []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		OnStart: func() {
			slog.Info("Ensuring resources are ready before starting...")

		},
		OnStarted: func() {
			slog.Info("Server started successfully")
			// You can add additional startup logic here
		},
		OnShutdown: func() {
			slog.Info("Cleanup before shutdown...")
			// Close database connections, etc.
		},
	}); err != nil {
		t.Error("Server error", "error", err)
	}

}

func TestCLI_LoadConfig(t *testing.T) {
	type TestConfig struct {
		DatabaseURL string `yaml:"database_url"`
		Debug       bool   `yaml:"debug"`
	}

	app := okapi.New()

	// Set up CLI flags
	cli := New(app, "Okapi Test").
		String("config", "c", "public/test_config.yaml", "Path to provider configuration file")

	err := cli.ParseFlags()
	if err != nil {
		t.Error(err)
	}
	restore := setOSArgs("--config", "public/test_config.yaml", "--port", "8080")
	defer restore()

	conf := &TestConfig{
		DatabaseURL: "postgres://user:pass@localhost:5432/dbname",
		Debug:       true,
	}
	err = os.MkdirAll("public", 0755)
	if err != nil {
		t.Error("Failed to create test directory", "error", err)
	}
	err = os.WriteFile("public/test_config.yaml", []byte("database_url: "+conf.DatabaseURL+"\ndebug: true\n"), 0644)
	if err != nil {
		t.Error("Failed to create test config file", "error", err)
	}
	configPath := cli.GetString("config")
	config := &TestConfig{}
	if err = cli.LoadConfig(configPath, config); err != nil {
		t.Error("Failed to load configuration", "error", err)
	}
	if config.DatabaseURL != "postgres://user:pass@localhost:5432/dbname" {
		t.Error("Unexpected DatabaseURL:", config.DatabaseURL)
	}
	if !config.Debug {
		t.Error("Expected Debug to be true")
	}
}
func TestCLI_FromStruct(t *testing.T) {
	o := okapi.New()
	config := &serverConfig{
		Port: 8000,
	}
	cli := New(o, "Okapi Test").FromStruct(config)
	err := cli.Parse()
	if err != nil {
		t.Error(err)
	}
	if config.Port != 8080 {
		t.Error("Expected default port 8080, got", config.Port)
	}
}
func TestCLI_WithConfig(t *testing.T) {
	o := okapi.New()
	config := &serverConfig{
		Port: 8000,
	}
	// Set APP_DEBUG env
	err := os.Setenv("APP_DEBUG", "true")
	if err != nil {
		t.Error("Failed to set environment variable", "error", err)
	}
	cli := New(o, "Okapi Test").WithConfig(config).MustParse()
	if config.Port != 8080 {
		t.Error("Expected default port 8080, got", config.Port)
	}
	// Check if debug flag is set from env
	if !cli.GetBool("debug") {
		t.Error("Expected debug to be true from environment variable")
	}
	// Check config debug value
	if !config.Debug {
		t.Error("Expected Debug to be true from environment variable")
	}

	port := cli.GetInt("port")
	if port != 8080 {
		t.Error("Expected port 8080, got", port)
	}
}

func TestCLI_Default(t *testing.T) {
	cli := Default().WithConfig(&serverConfig{Port: 8000}).MustParse()
	if cli.GetInt("port") != 8080 {
		t.Error("Expected default port 8080, got", cli.GetInt("port"))
	}
	if cli.Okapi() == nil {
		t.Error("Expected Okapi instance, got nil")
	}
	cli.Okapi().Get("/", func(c *okapi.Context) error {
		return c.String(200, "Hello, Okapi CLI!")
	})
}

func TestCLI_Command_Basic(t *testing.T) {
	app := okapi.New()
	cli := New(app, "test-app")

	var ran bool
	var gotPort int

	cli.Command("serve", "Start the server", func(cmd *Command) error {
		ran = true
		gotPort = cmd.GetInt("port")
		return nil
	}).Int("port", "p", 8080, "HTTP server port")

	restore := setOSArgs("serve", "--port", "9090")
	defer restore()

	if err := cli.Execute(); err != nil {
		t.Fatal("Execute failed:", err)
	}
	if !ran {
		t.Error("Expected serve command to run")
	}
	if gotPort != 9090 {
		t.Error("Expected port 9090, got", gotPort)
	}
	if cli.MatchedCommand() == nil || cli.MatchedCommand().Name() != "serve" {
		t.Error("Expected matched command to be 'serve'")
	}
}

func TestCLI_Command_FromStruct(t *testing.T) {
	type serveConfig struct {
		Port  int    `cli:"port"  short:"p" desc:"HTTP port" default:"8080"`
		Host  string `cli:"host"  short:"h" desc:"Hostname"  default:"localhost"`
		Debug bool   `cli:"debug" short:"d" desc:"Debug mode"`
	}

	app := okapi.New()
	cli := New(app, "test-app")

	cfg := &serveConfig{}
	cli.Command("serve", "Start the server", func(cmd *Command) error {
		return nil
	}).FromStruct(cfg)

	restore := setOSArgs("serve", "--port", "3000", "--debug")
	defer restore()

	if err := cli.Execute(); err != nil {
		t.Fatal("Execute failed:", err)
	}
	if cfg.Port != 3000 {
		t.Error("Expected port 3000, got", cfg.Port)
	}
	if cfg.Host != "localhost" {
		t.Error("Expected host localhost, got", cfg.Host)
	}
	if !cfg.Debug {
		t.Error("Expected debug to be true")
	}
}

func TestCLI_Command_EnvVars(t *testing.T) {
	type serveConfig struct {
		Port int `cli:"port" short:"p" desc:"HTTP port" env:"TEST_CMD_PORT" default:"8080"`
	}

	app := okapi.New()
	cli := New(app, "test-app")

	cfg := &serveConfig{}
	cli.Command("serve", "Start the server", func(cmd *Command) error {
		return nil
	}).FromStruct(cfg)

	if err := os.Setenv("TEST_CMD_PORT", "4000"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Unsetenv("TEST_CMD_PORT") }()

	restore := setOSArgs("serve")
	defer restore()

	if err := cli.Execute(); err != nil {
		t.Fatal("Execute failed:", err)
	}
	if cfg.Port != 4000 {
		t.Error("Expected port 4000 from env, got", cfg.Port)
	}
}

func TestCLI_Command_Unknown(t *testing.T) {
	app := okapi.New()
	cli := New(app, "test-app")

	cli.Command("serve", "Start the server", func(cmd *Command) error {
		return nil
	})

	restore := setOSArgs("unknown")
	defer restore()

	err := cli.Execute()
	if err == nil {
		t.Error("Expected error for unknown command")
	}
	if err != nil && !strings.Contains(err.Error(), "unknown command") {
		t.Error("Expected 'unknown command' error, got:", err)
	}
}

func TestCLI_Command_NoSubcommand(t *testing.T) {
	app := okapi.New()
	cli := New(app, "test-app")

	cli.Command("serve", "Start the server", func(cmd *Command) error {
		return nil
	})

	restore := setOSArgs()
	defer restore()

	err := cli.Execute()
	if err == nil {
		t.Error("Expected error when no subcommand specified")
	}
}

func TestCLI_Command_MultipleCommands(t *testing.T) {
	app := okapi.New()
	cli := New(app, "test-app")

	var serveRan, migrateRan bool

	cli.Command("serve", "Start the server", func(cmd *Command) error {
		serveRan = true
		return nil
	}).Int("port", "p", 8080, "HTTP port")

	cli.Command("migrate", "Run migrations", func(cmd *Command) error {
		migrateRan = true
		return nil
	}).String("dir", "d", "./migrations", "Migrations directory")

	restore := setOSArgs("migrate", "--dir", "/tmp/migrations")
	defer restore()

	if err := cli.Execute(); err != nil {
		t.Fatal("Execute failed:", err)
	}
	if serveRan {
		t.Error("serve should not have run")
	}
	if !migrateRan {
		t.Error("migrate should have run")
	}
}

func TestCLI_Command_Args(t *testing.T) {
	app := okapi.New()
	cli := New(app, "test-app")

	var gotArgs []string

	cli.Command("run", "Run a script", func(cmd *Command) error {
		gotArgs = cmd.Args()
		return nil
	}).Bool("verbose", "v", false, "Verbose output")

	restore := setOSArgs("run", "--verbose", "script.sh", "arg1", "arg2")
	defer restore()

	if err := cli.Execute(); err != nil {
		t.Fatal("Execute failed:", err)
	}
	if len(gotArgs) != 3 || gotArgs[0] != "script.sh" {
		t.Error("Expected args [script.sh arg1 arg2], got", gotArgs)
	}
}

func TestCLI_Command_OkapiAccess(t *testing.T) {
	app := okapi.New()
	cli := New(app, "test-app")

	cli.Command("serve", "Start the server", func(cmd *Command) error {
		if cmd.Okapi() != app {
			t.Error("Expected Okapi() to return the parent instance")
		}
		if cmd.CLI() != cli {
			t.Error("Expected CLI() to return the parent CLI")
		}
		return nil
	})

	restore := setOSArgs("serve")
	defer restore()

	if err := cli.Execute(); err != nil {
		t.Fatal("Execute failed:", err)
	}
}

func TestCLI_Execute_NoCommands_FallsBack(t *testing.T) {
	app := okapi.New()
	cli := New(app, "test-app").
		Int("port", "p", 8080, "HTTP port")

	restore := setOSArgs("--port", "3000")
	defer restore()

	// Execute with no commands registered should fall back to ParseFlags
	if err := cli.Execute(); err != nil {
		t.Fatal("Execute failed:", err)
	}
	if cli.GetInt("port") != 3000 {
		t.Error("Expected port 3000, got", cli.GetInt("port"))
	}
}

func TestCLI_Command_RunServer(t *testing.T) {
	app := okapi.New()
	cli := New(app, "test-app")

	cli.Command("serve", "Start the server", func(cmd *Command) error {
		port := cmd.GetInt("port")
		cmd.Okapi().WithPort(port)

		okapitest.GracefulExitAfter(3 * time.Second)
		return cmd.CLI().RunServer(&RunOptions{
			ShutdownTimeout: 10 * time.Second,
			OnStart: func() {
				slog.Info("Server starting", "port", port)
			},
		})
	}).Int("port", "p", 8080, "HTTP port")

	restore := setOSArgs("serve", "--port", "18923")
	defer restore()

	if err := cli.Execute(); err != nil {
		t.Fatal("Execute failed:", err)
	}
}
