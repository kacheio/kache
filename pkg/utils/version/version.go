// MIT License
//
// Copyright (c) 2023 kache.io
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package version

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"strings"
)

// Build information. Populated at build-time.
var (
	Version = "unknown"
	Build   = "unknown"
	Branch  = "unknown"
	Runtime = runtime.Version()
)

// versionTmpl is the version template.
var versionTmpl = `
{{.name}}, version {{.version}} (branch={{.branch}}, build={{.build}})
  runtime:       	{{.runtime}}
  platform:         {{.platform}}
`

// Print returns the version print.
func Print(name string) string {
	m := map[string]string{
		"name":     name,
		"version":  Version,
		"build":    Build,
		"branch":   Branch,
		"runtime":  Runtime,
		"platform": runtime.GOOS + "/" + runtime.GOARCH,
	}
	t := template.Must(template.New("version").Parse(versionTmpl))

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "version", m); err != nil {
		panic(err)
	}
	return strings.TrimSpace(buf.String())
}

// Info returns version info with version, branch, and build.
func Info() string {
	return fmt.Sprintf("[version=%s, branch=%s, build=%s]", Version, Branch, Build)
}

// Handler is the verson http handler func.
func Handler(w http.ResponseWriter, r *http.Request) {
	v := struct {
		Version  string
		Branch   string
		Build    string
		Runtime  string
		Platform string
	}{
		Version:  Version,
		Branch:   Branch,
		Build:    Build,
		Runtime:  Runtime,
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
