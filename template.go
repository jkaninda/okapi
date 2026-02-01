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
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"text/template"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, _ *Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// TemplateConfig holds configuration for template loading
type TemplateConfig struct {
	// File pattern (e.g., "views/*.html")
	Pattern string
	// Embedded or custom filesystem
	FS fs.FS
	// Custom template functions
	Funcs template.FuncMap
	// Base directory for templates
	BaseDir string
}

// NewTemplate creates a template from embedded filesystem
func NewTemplate(fsys fs.FS, pattern string) (*Template, error) {
	tmpl, err := template.ParseFS(fsys, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates from fs: %w", err)
	}
	return &Template{templates: tmpl}, nil
}

// NewTemplateFromFiles creates a template from file system with pattern
//
// Example:
//
//		tmpl, err := okapi.NewTemplateFromFiles("public/views/*.html")
//	 if err != nil {
//		 // handle error
//	 }
//		o := okapi.New().WithRenderer(tmpl)
func NewTemplateFromFiles(pattern string) (*Template, error) {
	tmpl, err := template.ParseGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template files: %w", err)
	}
	return &Template{templates: tmpl}, nil
}

// NewTemplateFromDirectory creates a template from a directory
//
// Example:
//
//		tmpl, err := okapi.NewTemplateFromDirectory("templates", ".html", ".tmpl")
//	 if err != nil {
//		 // handle error
//	 }
//		o := okapi.New().WithRenderer(tmpl)
func NewTemplateFromDirectory(dir string, extensions ...string) (*Template, error) {
	if len(extensions) == 0 {
		extensions = []string{".html", ".tmpl"}
	}

	patterns := make([]string, 0, len(extensions)*2)
	for _, ext := range extensions {
		patterns = append(patterns, filepath.Join(dir, "**/*"+ext))
		patterns = append(patterns, filepath.Join(dir, "*"+ext))
	}

	tmpl := template.New("")
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob pattern %s: %w", pattern, err)
		}

		for _, match := range matches {
			_, err = tmpl.ParseFiles(match)
			if err != nil {
				return nil, fmt.Errorf("failed to parse file %s: %w", match, err)
			}
		}
	}

	if len(tmpl.Templates()) == 0 {
		return nil, fmt.Errorf("no templates found in directory: %s", dir)
	}

	return &Template{templates: tmpl}, nil
}

// NewTemplateWithConfig creates a template using configuration
//
// Example:
// With configuration
//
//	tmpl, _ := NewTemplateWithConfig(TemplateConfig{
//				FS: os.DirFS("templates"),
//				Pattern: "**/*.html",
//				Funcs: template.FuncMap{"upper": strings.ToUpper},
//	})
//	 if err != nil {
//		 // handle error
//	 }
//		o := okapi.New().WithRenderer(tmpl)
func NewTemplateWithConfig(config TemplateConfig) (*Template, error) {
	var tmpl *template.Template
	var err error

	// Initialize with custom functions if provided
	if config.Funcs != nil {
		tmpl = template.New("").Funcs(config.Funcs)
	} else {
		tmpl = template.New("")
	}

	// Parse templates based on source
	if config.FS != nil {
		// Use provided filesystem (embedded or custom)
		tmpl, err = tmpl.ParseFS(config.FS, config.Pattern)
	} else if config.BaseDir != "" {
		// Use base directory with pattern
		pattern := filepath.Join(config.BaseDir, config.Pattern)
		tmpl, err = tmpl.ParseGlob(pattern)
	} else {
		// Use pattern directly
		tmpl, err = tmpl.ParseGlob(config.Pattern)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	if len(tmpl.Templates()) == 0 {
		return nil, fmt.Errorf("no templates found with config: %+v", config)
	}

	return &Template{templates: tmpl}, nil
}

// AddTemplate allows adding templates dynamically after creation
//
// Example:
//
//	err := tmpl.AddTemplate("newTemplate", "<h1>{{.Title}}</h1>")
//	if err != nil {
//		// handle error
//	}
func (t *Template) AddTemplate(name, content string) error {
	_, err := t.templates.New(name).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to add template %s: %w", name, err)
	}
	return nil
}

// AddTemplateFile adds a template from a file
//
// Example:
//
//	err := tmpl.AddTemplateFile("path/to/template.html")
//	if err != nil {
//		// handle error
//	}
func (t *Template) AddTemplateFile(filepath string) error {
	_, err := t.templates.ParseFiles(filepath)
	if err != nil {
		return fmt.Errorf("failed to add template file %s: %w", filepath, err)
	}
	return nil
}

// WithDefaultRenderer sets renderer from default file pattern
func (o *Okapi) WithDefaultRenderer(templatePath string) *Okapi {
	tmpl, err := NewTemplateFromFiles(templatePath)
	if err != nil {
		panic(fmt.Sprintf("failed to load templates: %v", err))
	}
	return o.WithRenderer(tmpl)
}

// WithRendererFromFS sets renderer from embedded filesystem
func (o *Okapi) WithRendererFromFS(fsys fs.FS, pattern string) *Okapi {
	tmpl, err := NewTemplate(fsys, pattern)
	if err != nil {
		panic(fmt.Sprintf("failed to load templates from fs: %v", err))
	}
	return o.WithRenderer(tmpl)
}

// WithRendererFromDirectory sets renderer from directory
func (o *Okapi) WithRendererFromDirectory(dir string, extensions ...string) *Okapi {
	tmpl, err := NewTemplateFromDirectory(dir, extensions...)
	if err != nil {
		panic(fmt.Sprintf("failed to load templates from directory: %v", err))
	}
	return o.WithRenderer(tmpl)
}

// WithRendererConfig sets renderer using configuration
func (o *Okapi) WithRendererConfig(config TemplateConfig) *Okapi {
	tmpl, err := NewTemplateWithConfig(config)
	if err != nil {
		panic(fmt.Sprintf("failed to load templates with config: %v", err))
	}
	return o.WithRenderer(tmpl)
}
