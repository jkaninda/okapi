//go:build !windows

/*
 *  MIT License
 *
 * Copyright (c) 2025 Jonas Kaninda
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

// These tests deliver real signals to the test process, which Windows cannot do.

package okapicli

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/jkaninda/okapi"
)

// TestCLI_RunServer_ReleasesSignalHandler runs RunServer in a child process
// against a port already in use, then checks SIGINT still kills the process.
func TestCLI_RunServer_ReleasesSignalHandler(t *testing.T) {
	if os.Getenv("OKAPICLI_SIGNAL_CHILD") == "1" {
		ln, err := net.Listen("tcp", ":0")
		if err != nil {
			fmt.Println("listen:", err)
			os.Exit(2)
		}
		defer func() { _ = ln.Close() }()
		app := okapi.New().WithPort(ln.Addr().(*net.TCPAddr).Port)
		if err := New(app, "Okapi Test").RunServer(); err == nil {
			fmt.Println("expected a startup error")
			os.Exit(2)
		}
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		time.Sleep(2 * time.Second)
		fmt.Println("survived SIGINT")
		os.Exit(3)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestCLI_RunServer_ReleasesSignalHandler$")
	cmd.Env = append(os.Environ(), "OKAPICLI_SIGNAL_CHILD=1")
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("child exited with %v, want death by SIGINT\n%s", err, out)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGINT {
		t.Fatalf("child exited with %v, want death by SIGINT\n%s", err, out)
	}
}

// TestCLI_RunServer_PartialOptions checks that zero fields in RunOptions keep
// their defaults, so in-flight requests survive a graceful shutdown.
func TestCLI_RunServer_PartialOptions(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	app := okapi.New().WithAddr(addr)
	app.Get("/slow", func(c *okapi.Context) error {
		time.Sleep(500 * time.Millisecond)
		return c.OK(okapi.M{"status": "done"})
	})

	type result struct {
		status int
		err    error
	}
	done := make(chan result, 1)
	go func() {
		deadline := time.Now().Add(5 * time.Second)
		for {
			conn, err := net.Dial("tcp", addr)
			if err == nil {
				_ = conn.Close()
				break
			}
			if time.Now().After(deadline) {
				done <- result{err: fmt.Errorf("server not ready: %w", err)}
				_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		time.AfterFunc(100*time.Millisecond, func() {
			_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		})
		resp, err := http.Get("http://" + addr + "/slow")
		if err != nil {
			done <- result{err: err}
			return
		}
		_ = resp.Body.Close()
		done <- result{status: resp.StatusCode}
	}()

	if err := New(app, "Okapi Test").RunServer(&RunOptions{
		OnStart: func() {},
	}); err != nil {
		t.Fatalf("RunServer: %v", err)
	}
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("in-flight request: %v", r.err)
		}
		if r.status != http.StatusOK {
			t.Errorf("status = %d, want 200", r.status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("in-flight request did not complete")
	}
}
