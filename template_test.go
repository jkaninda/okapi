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

package okapi

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
	"testing"
	"text/template"
)

//go:embed examples/template/views/*.html
var testViews embed.FS

func setupTestTemplates(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "okapi-templates-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test template files
	templates := map[string]string{
		"index.html": `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title></head>
<body><h1>{{.Message}}</h1></body>
</html>`,
		"user.html": `<div class="user">
    <span>{{.Name}}</span>
    <span>{{.Email}}</span>
</div>`,
		"admin/dashboard.html": `<div class="dashboard">
    <h2>Admin Dashboard</h2>
    <p>Welcome, {{.AdminName}}</p>
</div>`,
		"partial.tmpl": `<footer>{{.Footer}}</footer>`,
	}

	for path, content := range templates {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)

		if err = os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err = os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write template file %s: %v", path, err)
		}
	}

	return tmpDir
}

func cleanupTestTemplates(t *testing.T, dir string) {
	t.Helper()
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Failed to cleanup temp dir: %v", err)
	}
}

// TestNewTemplate tests creating templates from embedded filesystem
func TestNewTemplate(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "valid pattern",
			pattern: "examples/template/views/*.html",
			wantErr: false,
		},
		{
			name:    "invalid pattern",
			pattern: "nonexistent/*.html",
			wantErr: true,
		},
		{
			name:    "empty pattern",
			pattern: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTemplate(testViews, tt.pattern)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error for pattern %q, got nil", tt.pattern)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for pattern %q: %v", tt.pattern, err)
			}
		})
	}
}

// TestNewTemplateFromFiles tests creating templates from file patterns
func TestNewTemplateFromFiles(t *testing.T) {
	tmpDir := setupTestTemplates(t)
	defer cleanupTestTemplates(t, tmpDir)

	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "single file pattern",
			pattern: filepath.Join(tmpDir, "*.html"),
			wantErr: false,
		},
		{
			name:    "nested pattern",
			pattern: filepath.Join(tmpDir, "**/*.html"),
			wantErr: false,
		},
		{
			name:    "invalid pattern",
			pattern: "/nonexistent/*.html",
			wantErr: true,
		},
		{
			name:    "no matching files",
			pattern: filepath.Join(tmpDir, "*.xyz"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := NewTemplateFromFiles(tt.pattern)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tmpl == nil {
				t.Error("Expected template but got nil")
			}
		})
	}
}

// TestNewTemplateFromDirectory tests creating templates from directories
func TestNewTemplateFromDirectory(t *testing.T) {
	tmpDir := setupTestTemplates(t)
	defer cleanupTestTemplates(t, tmpDir)

	tests := []struct {
		name       string
		dir        string
		extensions []string
		wantErr    bool
		wantCount  int
	}{
		{
			name:       "default extensions",
			dir:        tmpDir,
			extensions: nil,
			wantErr:    false,
			wantCount:  4, // index.html, user.html, dashboard.html, partial.tmpl
		},
		{
			name:       "html only",
			dir:        tmpDir,
			extensions: []string{".html"},
			wantErr:    false,
			wantCount:  3,
		},
		{
			name:       "tmpl only",
			dir:        tmpDir,
			extensions: []string{".tmpl"},
			wantErr:    false,
			wantCount:  1,
		},
		{
			name:       "nonexistent directory",
			dir:        "/nonexistent",
			extensions: nil,
			wantErr:    true,
			wantCount:  0,
		},
		{
			name:       "empty directory",
			dir:        func() string { d, _ := os.MkdirTemp("", "empty-*"); return d }(),
			extensions: nil,
			wantErr:    true,
			wantCount:  0,
		},
	}

	for _, _tt := range tests {
		tt := _tt

		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "empty directory" {
				defer func(path string) {
					if err := os.RemoveAll(path); err != nil {
						t.Errorf("Failed to remove temp dir: %v", err)
					}
				}(tt.dir)
			}

			tmpl, err := NewTemplateFromDirectory(tt.dir, tt.extensions...)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tmpl == nil {
				t.Error("Expected template but got nil")
				return
			}

			count := len(tmpl.templates.Templates())
			if count < tt.wantCount {
				t.Errorf("Expected at least %d templates, got %d", tt.wantCount, count)
			}
		})
	}
}

// TestNewTemplateWithConfig tests creating templates with configuration
func TestNewTemplateWithConfig(t *testing.T) {
	tmpDir := setupTestTemplates(t)
	defer cleanupTestTemplates(t, tmpDir)

	customFuncs := template.FuncMap{
		"upper": func(s string) string {
			return "UPPER_" + s
		},
	}

	tests := []struct {
		name    string
		config  TemplateConfig
		wantErr bool
	}{
		{
			name: "with embedded fs",
			config: TemplateConfig{
				FS:      testViews,
				Pattern: "examples/template/views/*.html",
			},
			wantErr: false,
		},
		{
			name: "with base directory",
			config: TemplateConfig{
				BaseDir: tmpDir,
				Pattern: "*.html",
			},
			wantErr: false,
		},
		{
			name: "with custom functions",
			config: TemplateConfig{
				BaseDir: tmpDir,
				Pattern: "*.html",
				Funcs:   customFuncs,
			},
			wantErr: false,
		},
		{
			name: "with pattern only",
			config: TemplateConfig{
				Pattern: filepath.Join(tmpDir, "*.html"),
			},
			wantErr: false,
		},
		{
			name: "invalid pattern",
			config: TemplateConfig{
				Pattern: "/nonexistent/*.html",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := NewTemplateWithConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tmpl == nil {
				t.Error("Expected template but got nil")
			}
		})
	}
}

// TestTemplateRender tests rendering templates
func TestTemplateRender(t *testing.T) {
	tmpDir := setupTestTemplates(t)
	defer cleanupTestTemplates(t, tmpDir)

	tmpl, err := NewTemplateFromFiles(filepath.Join(tmpDir, "*.html"))
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	tests := []struct {
		name     string
		template string
		data     interface{}
		wantErr  bool
		contains string
	}{
		{
			name:     "render index template",
			template: "index.html",
			data: map[string]string{
				"Title":   "Test Page",
				"Message": "Hello World",
			},
			wantErr:  false,
			contains: "Hello World",
		},
		{
			name:     "render user template",
			template: "user.html",
			data: map[string]string{
				"Name":  "Jonas Kaninda",
				"Email": "jkaninda@mail.com",
			},
			wantErr:  false,
			contains: "Jonas Kaninda",
		},
		{
			name:     "nonexistent template",
			template: "nonexistent.html",
			data:     nil,
			wantErr:  true,
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.Render(&buf, tt.template, tt.data, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.contains != "" && !bytes.Contains(buf.Bytes(), []byte(tt.contains)) {
				t.Errorf("Expected output to contain %q, got: %s", tt.contains, buf.String())
			}
		})
	}
}

// TestAddTemplate tests adding templates dynamically
func TestAddTemplate(t *testing.T) {
	tmpDir := setupTestTemplates(t)
	defer cleanupTestTemplates(t, tmpDir)

	tmpl, err := NewTemplateFromFiles(filepath.Join(tmpDir, "index.html"))
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	tests := []struct {
		name         string
		templateName string
		content      string
		wantErr      bool
	}{
		{
			name:         "add valid template",
			templateName: "dynamic.html",
			content:      "<p>{{.Content}}</p>",
			wantErr:      false,
		},
		{
			name:         "add invalid template",
			templateName: "invalid.html",
			content:      "<p>{{.Content}",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = tmpl.AddTemplate(tt.templateName, tt.content)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify template was added
			var buf bytes.Buffer
			err = tmpl.Render(&buf, tt.templateName, map[string]string{"Content": "test"}, nil)
			if err != nil {
				t.Errorf("Failed to render added template: %v", err)
			}
		})
	}
}

// TestAddTemplateFile tests adding template files dynamically
func TestAddTemplateFile(t *testing.T) {
	tmpDir := setupTestTemplates(t)
	defer cleanupTestTemplates(t, tmpDir)

	tmpl, err := NewTemplateFromFiles(filepath.Join(tmpDir, "index.html"))
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	tests := []struct {
		name     string
		filepath string
		wantErr  bool
	}{
		{
			name:     "add existing file",
			filepath: filepath.Join(tmpDir, "user.html"),
			wantErr:  false,
		},
		{
			name:     "add nonexistent file",
			filepath: filepath.Join(tmpDir, "nonexistent.html"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tmpl.AddTemplateFile(tt.filepath)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestOkapiWithRendererMethods tests Okapi convenience methods
func TestOkapiWithRendererMethods(t *testing.T) {
	tmpDir := setupTestTemplates(t)
	defer cleanupTestTemplates(t, tmpDir)

	t.Run("WithDefaultRenderer", func(t *testing.T) {
		app := New()

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic but didn't get one")
			}
		}()

		app.WithDefaultRenderer("/nonexistent/*.html")
	})

	t.Run("WithRendererFromDirectory", func(t *testing.T) {
		app := New()
		result := app.WithRendererFromDirectory(tmpDir)

		if result == nil {
			t.Error("Expected app instance but got nil")
		}

		if app.renderer == nil {
			t.Error("Expected renderer to be set")
		}
	})

	t.Run("WithRendererConfig", func(t *testing.T) {
		app := New()
		result := app.WithRendererConfig(TemplateConfig{
			BaseDir: tmpDir,
			Pattern: "*.html",
		})

		if result == nil {
			t.Error("Expected app instance but got nil")
		}

		if app.renderer == nil {
			t.Error("Expected renderer to be set")
		}
	})
}
