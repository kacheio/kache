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

package api

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"

	"github.com/gorilla/mux"
)

func init() {
	expvar.Publish("Goroutines2", expvar.Func(goroutines))
}

func goroutines() interface{} {
	return runtime.NumGoroutine()
}

// DebugHandler expose debug routes.
type DebugHandler struct{}

// Append add debug routes on a router.
func (g DebugHandler) Append(router *mux.Router) {
	router.Methods(http.MethodGet).Path("/debug/vars").
		HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprint(w, "{\n")
			first := true
			expvar.Do(func(kv expvar.KeyValue) {
				if !first {
					fmt.Fprint(w, ",\n")
				}
				first = false
				fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
			})
			fmt.Fprint(w, "\n}\n")
		})

	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(5)
	router.Methods(http.MethodGet).PathPrefix("/debug/pprof/cmdline").HandlerFunc(pprof.Cmdline)
	router.Methods(http.MethodGet).PathPrefix("/debug/pprof/profile").HandlerFunc(pprof.Profile)
	router.Methods(http.MethodGet).PathPrefix("/debug/pprof/symbol").HandlerFunc(pprof.Symbol)
	router.Methods(http.MethodGet).PathPrefix("/debug/pprof/trace").HandlerFunc(pprof.Trace)
	router.Methods(http.MethodGet).PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
}
