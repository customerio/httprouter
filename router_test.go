// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package httprouter

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type mockResponseWriter struct{}

func (m *mockResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *mockResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *mockResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *mockResponseWriter) WriteHeader(int) {}

func TestParams(t *testing.T) {
	ps := Params{
		Param{"param1", "value1"},
		Param{"param2", "value2"},
		Param{"param3", "value3"},
	}
	for i := range ps {
		if val := ps.ByName(ps[i].Key); val != ps[i].Value {
			t.Errorf("Wrong value for %s: Got %s; Want %s", ps[i].Key, val, ps[i].Value)
		}
	}
	if val := ps.ByName("noKey"); val != "" {
		t.Errorf("Expected empty string for not found key; got: %s", val)
	}
}

func TestRouter(t *testing.T) {
	router := New()

	routed := false
	router.Handle(http.MethodGet, "/user/:name", func(w http.ResponseWriter, r *http.Request, ps Params) {
		routed = true
		want := Params{Param{"name", "gopher"}}
		if !reflect.DeepEqual(ps, want) {
			t.Fatalf("wrong wildcard values: want %v, got %v", want, ps)
		}
	})

	w := new(mockResponseWriter)

	req, _ := http.NewRequest(http.MethodGet, "/user/gopher", nil)
	router.ServeHTTP(w, req)

	if !routed {
		t.Fatal("routing failed")
	}
}

type handlerStruct struct {
	handled *bool
}

func (h handlerStruct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	*h.handled = true
}

func TestRouterAPI(t *testing.T) {
	var get, head, options, post, put, patch, delete, handler, handlerFunc bool

	httpHandler := handlerStruct{&handler}

	router := New()
	router.GET("/GET", func(w http.ResponseWriter, r *http.Request, _ Params) {
		get = true
	})
	router.HEAD("/GET", func(w http.ResponseWriter, r *http.Request, _ Params) {
		head = true
	})
	router.OPTIONS("/GET", func(w http.ResponseWriter, r *http.Request, _ Params) {
		options = true
	})
	router.POST("/POST", func(w http.ResponseWriter, r *http.Request, _ Params) {
		post = true
	})
	router.PUT("/PUT", func(w http.ResponseWriter, r *http.Request, _ Params) {
		put = true
	})
	router.PATCH("/PATCH", func(w http.ResponseWriter, r *http.Request, _ Params) {
		patch = true
	})
	router.DELETE("/DELETE", func(w http.ResponseWriter, r *http.Request, _ Params) {
		delete = true
	})
	router.Handler(http.MethodGet, "/Handler", httpHandler)
	router.HandlerFunc(http.MethodGet, "/HandlerFunc", func(w http.ResponseWriter, r *http.Request) {
		handlerFunc = true
	})

	w := new(mockResponseWriter)

	r, _ := http.NewRequest(http.MethodGet, "/GET", nil)
	router.ServeHTTP(w, r)
	if !get {
		t.Error("routing GET failed")
	}

	r, _ = http.NewRequest(http.MethodHead, "/GET", nil)
	router.ServeHTTP(w, r)
	if !head {
		t.Error("routing HEAD failed")
	}

	r, _ = http.NewRequest(http.MethodOptions, "/GET", nil)
	router.ServeHTTP(w, r)
	if !options {
		t.Error("routing OPTIONS failed")
	}

	r, _ = http.NewRequest(http.MethodPost, "/POST", nil)
	router.ServeHTTP(w, r)
	if !post {
		t.Error("routing POST failed")
	}

	r, _ = http.NewRequest(http.MethodPut, "/PUT", nil)
	router.ServeHTTP(w, r)
	if !put {
		t.Error("routing PUT failed")
	}

	r, _ = http.NewRequest(http.MethodPatch, "/PATCH", nil)
	router.ServeHTTP(w, r)
	if !patch {
		t.Error("routing PATCH failed")
	}

	r, _ = http.NewRequest(http.MethodDelete, "/DELETE", nil)
	router.ServeHTTP(w, r)
	if !delete {
		t.Error("routing DELETE failed")
	}

	r, _ = http.NewRequest(http.MethodGet, "/Handler", nil)
	router.ServeHTTP(w, r)
	if !handler {
		t.Error("routing Handler failed")
	}

	r, _ = http.NewRequest(http.MethodGet, "/HandlerFunc", nil)
	router.ServeHTTP(w, r)
	if !handlerFunc {
		t.Error("routing HandlerFunc failed")
	}
}

func TestRouterInvalidInput(t *testing.T) {
	router := New()

	handle := func(_ http.ResponseWriter, _ *http.Request, _ Params) {}

	recv := catchPanic(func() {
		router.Handle("", "/", handle)
	})
	if recv == nil {
		t.Fatal("registering empty method did not panic")
	}

	recv = catchPanic(func() {
		router.GET("", handle)
	})
	if recv == nil {
		t.Fatal("registering empty path did not panic")
	}

	recv = catchPanic(func() {
		router.GET("noSlashRoot", handle)
	})
	if recv == nil {
		t.Fatal("registering path not beginning with '/' did not panic")
	}

	recv = catchPanic(func() {
		router.GET("/", nil)
	})
	if recv == nil {
		t.Fatal("registering nil handler did not panic")
	}
}

func TestRouterChaining(t *testing.T) {
	router1 := New()
	router2 := New()
	router1.NotFound = router2

	fooHit := false
	router1.POST("/foo", func(w http.ResponseWriter, req *http.Request, _ Params) {
		fooHit = true
		w.WriteHeader(http.StatusOK)
	})

	barHit := false
	router2.POST("/bar", func(w http.ResponseWriter, req *http.Request, _ Params) {
		barHit = true
		w.WriteHeader(http.StatusOK)
	})

	r, _ := http.NewRequest(http.MethodPost, "/foo", nil)
	w := httptest.NewRecorder()
	router1.ServeHTTP(w, r)
	if !(w.Code == http.StatusOK && fooHit) {
		t.Errorf("Regular routing failed with router chaining.")
		t.FailNow()
	}

	r, _ = http.NewRequest(http.MethodPost, "/bar", nil)
	w = httptest.NewRecorder()
	router1.ServeHTTP(w, r)
	if !(w.Code == http.StatusOK && barHit) {
		t.Errorf("Chained routing failed with router chaining.")
		t.FailNow()
	}

	r, _ = http.NewRequest(http.MethodPost, "/qax", nil)
	w = httptest.NewRecorder()
	router1.ServeHTTP(w, r)
	if !(w.Code == http.StatusNotFound) {
		t.Errorf("NotFound behavior failed with router chaining.")
		t.FailNow()
	}
}

func BenchmarkAllowed(b *testing.B) {
	handlerFunc := func(_ http.ResponseWriter, _ *http.Request, _ Params) {}

	router := New()
	router.POST("/path", handlerFunc)
	router.GET("/path", handlerFunc)

	b.Run("Global", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = router.allowed("*", http.MethodOptions)
		}
	})
	b.Run("Path", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = router.allowed("/path", http.MethodOptions)
		}
	})
}

func TestRouterOPTIONS(t *testing.T) {
	handlerFunc := func(_ http.ResponseWriter, _ *http.Request, _ Params) {}

	router := New()
	router.POST("/path", handlerFunc)

	// test not allowed
	// * (server)
	r, _ := http.NewRequest(http.MethodOptions, "*", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// path
	r, _ = http.NewRequest(http.MethodOptions, "/path", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	r, _ = http.NewRequest(http.MethodOptions, "/doesnotexist", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNotFound) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	}

	// add another method
	router.GET("/path", handlerFunc)

	// set a global OPTIONS handler
	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Adjust status code to 204
		w.WriteHeader(http.StatusNoContent)
	})

	// test again
	// * (server)
	r, _ = http.NewRequest(http.MethodOptions, "*", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNoContent) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "GET, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// path
	r, _ = http.NewRequest(http.MethodOptions, "/path", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNoContent) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "GET, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// custom handler
	var custom bool
	router.OPTIONS("/path", func(w http.ResponseWriter, r *http.Request, _ Params) {
		custom = true
	})

	// test again
	// * (server)
	r, _ = http.NewRequest(http.MethodOptions, "*", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNoContent) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "GET, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}
	if custom {
		t.Error("custom handler called on *")
	}

	// path
	r, _ = http.NewRequest(http.MethodOptions, "/path", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, w.Header())
	}
	if !custom {
		t.Error("custom handler not called")
	}
}

func TestRouterNotAllowed(t *testing.T) {
	handlerFunc := func(_ http.ResponseWriter, _ *http.Request, _ Params) {}

	router := New()
	router.POST("/path", handlerFunc)

	// test not allowed
	r, _ := http.NewRequest(http.MethodGet, "/path", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusMethodNotAllowed) {
		t.Errorf("NotAllowed handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// add another method
	router.DELETE("/path", handlerFunc)
	router.OPTIONS("/path", handlerFunc) // must be ignored

	// test again
	r, _ = http.NewRequest(http.MethodGet, "/path", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusMethodNotAllowed) {
		t.Errorf("NotAllowed handling failed: Code=%d, Header=%v", w.Code, w.Header())
	} else if allow := w.Header().Get("Allow"); allow != "DELETE, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// test custom handler
	w = httptest.NewRecorder()
	responseText := "custom method"
	router.MethodNotAllowed = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte(responseText))
	})
	router.ServeHTTP(w, r)
	if got := w.Body.String(); !(got == responseText) {
		t.Errorf("unexpected response got %q want %q", got, responseText)
	}
	if w.Code != http.StatusTeapot {
		t.Errorf("unexpected response code %d want %d", w.Code, http.StatusTeapot)
	}
	if allow := w.Header().Get("Allow"); allow != "DELETE, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}
}

func TestRouterNotFound(t *testing.T) {
	handlerFunc := func(_ http.ResponseWriter, _ *http.Request, _ Params) {}

	router := New()
	router.GET("/path", handlerFunc)
	router.GET("/dir/", handlerFunc)
	router.GET("/", handlerFunc)

	testRoutes := []struct {
		route    string
		code     int
		location string
	}{
		{"/path/", http.StatusMovedPermanently, "/path"},   // TSR -/
		{"/dir", http.StatusMovedPermanently, "/dir/"},     // TSR +/
		{"", http.StatusMovedPermanently, "/"},             // TSR +/
		{"/PATH", http.StatusMovedPermanently, "/path"},    // Fixed Case
		{"/DIR/", http.StatusMovedPermanently, "/dir/"},    // Fixed Case
		{"/PATH/", http.StatusMovedPermanently, "/path"},   // Fixed Case -/
		{"/DIR", http.StatusMovedPermanently, "/dir/"},     // Fixed Case +/
		{"/../path", http.StatusMovedPermanently, "/path"}, // CleanPath
		{"/nope", http.StatusNotFound, ""},                 // NotFound
	}
	for _, tr := range testRoutes {
		r, _ := http.NewRequest(http.MethodGet, tr.route, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if !(w.Code == tr.code && (w.Code == http.StatusNotFound || fmt.Sprint(w.Header().Get("Location")) == tr.location)) {
			t.Errorf("NotFound handling route %s failed: Code=%d, Header=%v", tr.route, w.Code, w.Header().Get("Location"))
		}
	}

	// Test custom not found handler
	var notFound bool
	router.NotFound = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
		notFound = true
	})
	r, _ := http.NewRequest(http.MethodGet, "/nope", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNotFound && notFound == true) {
		t.Errorf("Custom NotFound handler failed: Code=%d, Header=%v", w.Code, w.Header())
	}

	// Test other method than GET (want 308 instead of 301)
	router.PATCH("/path", handlerFunc)
	r, _ = http.NewRequest(http.MethodPatch, "/path/", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusPermanentRedirect && fmt.Sprint(w.Header()) == "map[Location:[/path]]") {
		t.Errorf("Custom NotFound handler failed: Code=%d, Header=%v", w.Code, w.Header())
	}

	// Test special case where no node for the prefix "/" exists
	router = New()
	router.GET("/a", handlerFunc)
	r, _ = http.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNotFound) {
		t.Errorf("NotFound handling route / failed: Code=%d", w.Code)
	}
}

func TestRouterPanicHandler(t *testing.T) {
	router := New()
	panicHandled := false

	router.PanicHandler = func(rw http.ResponseWriter, r *http.Request, p interface{}) {
		panicHandled = true
	}

	router.Handle(http.MethodPut, "/user/:name", func(_ http.ResponseWriter, _ *http.Request, _ Params) {
		panic("oops!")
	})

	w := new(mockResponseWriter)
	req, _ := http.NewRequest(http.MethodPut, "/user/gopher", nil)

	defer func() {
		if rcv := recover(); rcv != nil {
			t.Fatal("handling panic failed")
		}
	}()

	router.ServeHTTP(w, req)

	if !panicHandled {
		t.Fatal("simulating failed")
	}
}

func TestRouterLookup(t *testing.T) {
	routed := false
	wantHandle := func(_ http.ResponseWriter, _ *http.Request, _ Params) {
		routed = true
	}
	wantParams := Params{Param{"name", "gopher"}}

	router := New()

	// try empty router first
	handle, _, tsr := router.Lookup(http.MethodGet, "/nope")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}

	// insert route and try again
	router.GET("/user/:name", wantHandle)
	handle, params, _ := router.Lookup(http.MethodGet, "/user/gopher")
	if handle == nil {
		t.Fatal("Got no handle!")
	} else {
		handle(nil, nil, nil)
		if !routed {
			t.Fatal("Routing failed!")
		}
	}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("Wrong parameter values: want %v, got %v", wantParams, params)
	}
	routed = false

	// route without param
	router.GET("/user", wantHandle)
	handle, params, _ = router.Lookup(http.MethodGet, "/user")
	if handle == nil {
		t.Fatal("Got no handle!")
	} else {
		handle(nil, nil, nil)
		if !routed {
			t.Fatal("Routing failed!")
		}
	}
	if params != nil {
		t.Fatalf("Wrong parameter values: want %v, got %v", nil, params)
	}

	handle, _, tsr = router.Lookup(http.MethodGet, "/user/gopher/")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if !tsr {
		t.Error("Got no TSR recommendation!")
	}

	handle, _, tsr = router.Lookup(http.MethodGet, "/nope")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}
}

func TestRouterParamsFromContext(t *testing.T) {
	routed := false

	wantParams := Params{Param{"name", "gopher"}}
	handlerFunc := func(_ http.ResponseWriter, req *http.Request) {
		// get params from request context
		params := ParamsFromContext(req.Context())

		if !reflect.DeepEqual(params, wantParams) {
			t.Fatalf("Wrong parameter values: want %v, got %v", wantParams, params)
		}

		routed = true
	}

	var nilParams Params
	handlerFuncNil := func(_ http.ResponseWriter, req *http.Request) {
		// get params from request context
		params := ParamsFromContext(req.Context())

		if !reflect.DeepEqual(params, nilParams) {
			t.Fatalf("Wrong parameter values: want %v, got %v", nilParams, params)
		}

		routed = true
	}
	router := New()
	router.HandlerFunc(http.MethodGet, "/user", handlerFuncNil)
	router.HandlerFunc(http.MethodGet, "/user/:name", handlerFunc)

	w := new(mockResponseWriter)
	r, _ := http.NewRequest(http.MethodGet, "/user/gopher", nil)
	router.ServeHTTP(w, r)
	if !routed {
		t.Fatal("Routing failed!")
	}

	routed = false
	r, _ = http.NewRequest(http.MethodGet, "/user", nil)
	router.ServeHTTP(w, r)
	if !routed {
		t.Fatal("Routing failed!")
	}
}

func TestRouterMatchedRoutePath(t *testing.T) {
	route1 := "/user/:name"
	routed1 := false
	handle1 := func(_ http.ResponseWriter, req *http.Request, ps Params) {
		route := ps.MatchedRoutePath()
		if route != route1 {
			t.Fatalf("Wrong matched route: want %s, got %s", route1, route)
		}
		routed1 = true
	}

	route2 := "/user/:name/details"
	routed2 := false
	handle2 := func(_ http.ResponseWriter, req *http.Request, ps Params) {
		route := ps.MatchedRoutePath()
		if route != route2 {
			t.Fatalf("Wrong matched route: want %s, got %s", route2, route)
		}
		routed2 = true
	}

	route3 := "/"
	routed3 := false
	handle3 := func(_ http.ResponseWriter, req *http.Request, ps Params) {
		route := ps.MatchedRoutePath()
		if route != route3 {
			t.Fatalf("Wrong matched route: want %s, got %s", route3, route)
		}
		routed3 = true
	}

	router := New()
	router.SaveMatchedRoutePath = true
	router.Handle(http.MethodGet, route1, handle1)
	router.Handle(http.MethodGet, route2, handle2)
	router.Handle(http.MethodGet, route3, handle3)

	w := new(mockResponseWriter)
	r, _ := http.NewRequest(http.MethodGet, "/user/gopher", nil)
	router.ServeHTTP(w, r)
	if !routed1 || routed2 || routed3 {
		t.Fatal("Routing failed!")
	}

	w = new(mockResponseWriter)
	r, _ = http.NewRequest(http.MethodGet, "/user/gopher/details", nil)
	router.ServeHTTP(w, r)
	if !routed2 || routed3 {
		t.Fatal("Routing failed!")
	}

	w = new(mockResponseWriter)
	r, _ = http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, r)
	if !routed3 {
		t.Fatal("Routing failed!")
	}
}

type mockFileSystem struct {
	opened bool
}

func (mfs *mockFileSystem) Open(name string) (http.File, error) {
	mfs.opened = true
	return nil, errors.New("this is just a mock")
}

func TestRouterServeFiles(t *testing.T) {
	router := New()
	mfs := &mockFileSystem{}

	recv := catchPanic(func() {
		router.ServeFiles("/noFilepath", mfs)
	})
	if recv == nil {
		t.Fatal("registering path not ending with '*filepath' did not panic")
	}

	router.ServeFiles("/*filepath", mfs)
	w := new(mockResponseWriter)
	r, _ := http.NewRequest(http.MethodGet, "/favicon.ico", nil)
	router.ServeHTTP(w, r)
	if !mfs.opened {
		t.Error("serving file failed")
	}
}

func TestRouterURLEncoding_Issue106_CurrentBehavior(t *testing.T) {
	// This test documents the CURRENT behavior of the router regarding URL encoding
	// Issue #106 was about preserving URL encoding, but the current implementation
	// still unescapes parameter values even when using RequestURI

	router := New()

	var capturedParam string
	router.GET("/user/:name", func(w http.ResponseWriter, r *http.Request, ps Params) {
		capturedParam = ps.ByName("name")
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name                 string
		path                 string
		expectedDecodedParam string // What we currently get (decoded)
		expectedRawParam     string // What Issue #106 intended (raw/encoded)
	}{
		{
			name:                 "encoded forward slash",
			path:                 "/user/john%2Fdoe",
			expectedDecodedParam: "john/doe",   // Current behavior: decoded by pathUnescape()
			expectedRawParam:     "john%2Fdoe", // Issue #106 intent: preserve encoding
		},
		{
			name:                 "encoded space",
			path:                 "/user/john%20doe",
			expectedDecodedParam: "john doe",   // Current: decoded
			expectedRawParam:     "john%20doe", // Issue #106: preserve encoding
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedParam = ""

			// Test with RequestURI populated (production-like)
			req, _ := http.NewRequest(http.MethodGet, tt.path, nil)
			req.RequestURI = tt.path

			t.Logf("Request URL.Path: %q (decoded by Go)", req.URL.Path)
			t.Logf("Request RequestURI: %q (raw, as sent by client)", req.RequestURI)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// Document current behavior
			if capturedParam == tt.expectedDecodedParam {
				t.Logf("✓ Current behavior confirmed: parameter = %q (decoded)", capturedParam)
			} else {
				t.Errorf("Unexpected current behavior: got %q, expected decoded value %q", capturedParam, tt.expectedDecodedParam)
			}

			// Show what Issue #106 intended
			t.Logf("⚠️  Issue #106 intent: parameter should be %q (preserve encoding)", tt.expectedRawParam)
			t.Logf("📝 Gap: tree.getValue() calls pathUnescape() on extracted parameters")
		})
	}
}

func TestRouterURLEncoding_FallbackCompatibility(t *testing.T) {
	// This test validates our fallback mechanism works for test environments
	// where RequestURI is empty (like http.NewRequest creates)
	router := New()

	var captured string
	router.GET("/simple/:param", func(w http.ResponseWriter, r *http.Request, ps Params) {
		captured = ps.ByName("param")
		w.WriteHeader(http.StatusOK)
	})

	t.Run("fallback to URL.Path when RequestURI empty", func(t *testing.T) {
		captured = ""

		// This simulates test environment - RequestURI is empty
		req, _ := http.NewRequest(http.MethodGet, "/simple/testvalue", nil)
		// req.RequestURI is empty by default from http.NewRequest

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if captured != "testvalue" {
			t.Errorf("Expected parameter 'testvalue', got %q", captured)
		}
	})

	t.Run("prefers RequestURI when available", func(t *testing.T) {
		captured = ""

		// This simulates production environment - RequestURI is populated
		req, _ := http.NewRequest(http.MethodGet, "/simple/testvalue", nil)
		req.RequestURI = "/simple/encoded%20value" // Different from URL.Path

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Current behavior: even when using RequestURI, parameters are decoded by pathUnescape()
		if captured != "encoded value" {
			t.Errorf("Expected parameter 'encoded value' (decoded), got %q", captured)
		}

		t.Logf("✓ RequestURI path used for routing: %q", req.RequestURI)
		t.Logf("✓ But parameter still decoded: %q", captured)
		t.Logf("📝 This confirms our fallback mechanism works for routing")
	})
}

func TestRouterURLEncoding_Issue106_BreakageDemo(t *testing.T) {
	// This test demonstrates the limitation when only URL.Path is available
	// (i.e., what happens in test environments without RequestURI)

	t.Run("demonstrate_critical_routing_breakage", func(t *testing.T) {
		router := New()

		var captured string
		router.GET("/user/:name", func(w http.ResponseWriter, r *http.Request, ps Params) {
			captured = ps.ByName("name")
			w.WriteHeader(http.StatusOK)
		})

		// This demonstrates the CRITICAL Issue #106 problem
		req, _ := http.NewRequest(http.MethodGet, "/user/john%2Fdoe", nil)

		t.Logf("🚨 CRITICAL Issue #106 Problem Demonstration:")
		t.Logf("   Original URL: /user/john%%2Fdoe")
		t.Logf("   Route pattern: /user/:name")
		t.Logf("   req.URL.Path: %q (decoded by Go)", req.URL.Path)
		t.Logf("   req.RequestURI: %q (empty in tests)", req.RequestURI)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// This SHOULD be 404 because URL.Path breaks the routing!
		if w.Code == http.StatusNotFound {
			t.Logf("✓ Expected 404 - routing broke due to URL decoding")
			t.Logf("💥 Problem: /user/john%%2Fdoe → /user/john/doe (2 segments, doesn't match :name)")
		} else {
			t.Errorf("Expected 404 (routing failure), got %d", w.Code)
		}

		t.Logf("📝 This is EXACTLY why Issue #106 needed RequestURI:")
		t.Logf("   - URL.Path decodes %%2F to /, breaking route matching")
		t.Logf("   - RequestURI preserves %%2F, allowing route to match")
		t.Logf("   - Without RequestURI, encoded slashes in parameters break routing")

		// Now test what happens WITH RequestURI (production scenario)
		req.RequestURI = "/user/john%2Fdoe" // Simulate production
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			t.Logf("✅ With RequestURI: routing works! Status=%d", w.Code)
			t.Logf("✅ Parameter captured: %q (decoded but route matched)", captured)
		} else {
			t.Errorf("With RequestURI, expected 200, got %d", w.Code)
		}
	})
}

func TestRouterMiddleware(t *testing.T) {
	// Test basic middleware functionality
	t.Run("basic middleware", func(t *testing.T) {
		router := New()
		middlewareCalled := false
		handlerCalled := false

		// Add middleware that sets a flag
		router.Use(func(next Handle) Handle {
			return func(w http.ResponseWriter, r *http.Request, ps Params) {
				middlewareCalled = true
				next(w, r, ps)
			}
		})

		// Add a route
		router.GET("/test", func(w http.ResponseWriter, r *http.Request, ps Params) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		// Make request
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if !middlewareCalled {
			t.Error("Middleware was not called")
		}
		if !handlerCalled {
			t.Error("Handler was not called")
		}
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Test multiple middleware chaining
	t.Run("multiple middleware chaining", func(t *testing.T) {
		router := New()
		var callOrder []string

		// Add first middleware
		router.Use(func(next Handle) Handle {
			return func(w http.ResponseWriter, r *http.Request, ps Params) {
				callOrder = append(callOrder, "middleware1-before")
				next(w, r, ps)
				callOrder = append(callOrder, "middleware1-after")
			}
		})

		// Add second middleware
		router.Use(func(next Handle) Handle {
			return func(w http.ResponseWriter, r *http.Request, ps Params) {
				callOrder = append(callOrder, "middleware2-before")
				next(w, r, ps)
				callOrder = append(callOrder, "middleware2-after")
			}
		})

		// Add handler
		router.GET("/chain", func(w http.ResponseWriter, r *http.Request, ps Params) {
			callOrder = append(callOrder, "handler")
			w.WriteHeader(http.StatusOK)
		})

		// Make request
		req, _ := http.NewRequest(http.MethodGet, "/chain", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify execution order (first added middleware runs outermost)
		expectedOrder := []string{
			"middleware1-before", // First added middleware runs outermost
			"middleware2-before",
			"handler",
			"middleware2-after",
			"middleware1-after", // First added middleware finishes outermost
		}

		if len(callOrder) != len(expectedOrder) {
			t.Errorf("Expected %d calls, got %d", len(expectedOrder), len(callOrder))
		}

		for i, expected := range expectedOrder {
			if i >= len(callOrder) || callOrder[i] != expected {
				t.Errorf("Call order mismatch at position %d: expected %q, got %q", i, expected, callOrder[i])
			}
		}
	})

	// Test middleware with request modification
	t.Run("middleware modifying request", func(t *testing.T) {
		router := New()

		// Middleware that adds a header
		router.Use(func(next Handle) Handle {
			return func(w http.ResponseWriter, r *http.Request, ps Params) {
				r.Header.Set("X-Middleware", "applied")
				next(w, r, ps)
			}
		})

		var receivedHeader string
		router.GET("/modify", func(w http.ResponseWriter, r *http.Request, ps Params) {
			receivedHeader = r.Header.Get("X-Middleware")
			w.WriteHeader(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/modify", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if receivedHeader != "applied" {
			t.Errorf("Expected header 'applied', got %q", receivedHeader)
		}
	})

	// Test middleware with response modification
	t.Run("middleware modifying response", func(t *testing.T) {
		router := New()

		// Middleware that adds response header
		router.Use(func(next Handle) Handle {
			return func(w http.ResponseWriter, r *http.Request, ps Params) {
				w.Header().Set("X-Custom-Header", "middleware-value")
				next(w, r, ps)
			}
		})

		router.GET("/response", func(w http.ResponseWriter, r *http.Request, ps Params) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("response body"))
		})

		req, _ := http.NewRequest(http.MethodGet, "/response", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Header().Get("X-Custom-Header") != "middleware-value" {
			t.Error("Middleware did not set response header")
		}
		if w.Body.String() != "response body" {
			t.Errorf("Expected body 'response body', got %q", w.Body.String())
		}
	})

	// Test middleware with parameters
	t.Run("middleware with parameters", func(t *testing.T) {
		router := New()
		var capturedParams Params

		// Middleware that captures params
		router.Use(func(next Handle) Handle {
			return func(w http.ResponseWriter, r *http.Request, ps Params) {
				capturedParams = ps
				next(w, r, ps)
			}
		})

		router.GET("/user/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request, ps Params) {
			w.WriteHeader(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/user/123/posts/456", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if len(capturedParams) != 2 {
			t.Errorf("Expected 2 params, got %d", len(capturedParams))
		}
		if capturedParams.ByName("id") != "123" {
			t.Errorf("Expected id=123, got %q", capturedParams.ByName("id"))
		}
		if capturedParams.ByName("postId") != "456" {
			t.Errorf("Expected postId=456, got %q", capturedParams.ByName("postId"))
		}
	})

	// Test middleware affecting different HTTP methods
	t.Run("middleware with different HTTP methods", func(t *testing.T) {
		router := New()
		var methodsSeen []string

		// Middleware that tracks HTTP methods
		router.Use(func(next Handle) Handle {
			return func(w http.ResponseWriter, r *http.Request, ps Params) {
				methodsSeen = append(methodsSeen, r.Method)
				next(w, r, ps)
			}
		})

		// Register handlers for different methods
		router.GET("/api", func(w http.ResponseWriter, r *http.Request, ps Params) {
			w.WriteHeader(http.StatusOK)
		})
		router.POST("/api", func(w http.ResponseWriter, r *http.Request, ps Params) {
			w.WriteHeader(http.StatusCreated)
		})
		router.PUT("/api", func(w http.ResponseWriter, r *http.Request, ps Params) {
			w.WriteHeader(http.StatusOK)
		})

		// Test GET
		req, _ := http.NewRequest(http.MethodGet, "/api", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("GET: Expected status 200, got %d", w.Code)
		}

		// Test POST
		req, _ = http.NewRequest(http.MethodPost, "/api", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("POST: Expected status 201, got %d", w.Code)
		}

		// Test PUT
		req, _ = http.NewRequest(http.MethodPut, "/api", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("PUT: Expected status 200, got %d", w.Code)
		}

		expectedMethods := []string{"GET", "POST", "PUT"}
		if len(methodsSeen) != len(expectedMethods) {
			t.Errorf("Expected %d method calls, got %d", len(expectedMethods), len(methodsSeen))
		}
		for i, expected := range expectedMethods {
			if i >= len(methodsSeen) || methodsSeen[i] != expected {
				t.Errorf("Method %d: expected %s, got %s", i, expected, methodsSeen[i])
			}
		}
	})

	// Test early return from middleware
	t.Run("middleware early return", func(t *testing.T) {
		router := New()
		handlerCalled := false

		// Middleware that returns early under certain conditions
		router.Use(func(next Handle) Handle {
			return func(w http.ResponseWriter, r *http.Request, ps Params) {
				if r.Header.Get("X-Block") == "true" {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("blocked by middleware"))
					return // Don't call next()
				}
				next(w, r, ps)
			}
		})

		router.GET("/protected", func(w http.ResponseWriter, r *http.Request, ps Params) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("handler response"))
		})

		// Test blocked request
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("X-Block", "true")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
		if w.Body.String() != "blocked by middleware" {
			t.Errorf("Expected 'blocked by middleware', got %q", w.Body.String())
		}
		if handlerCalled {
			t.Error("Handler should not have been called when middleware blocks")
		}

		// Test allowed request
		handlerCalled = false
		req, _ = http.NewRequest(http.MethodGet, "/protected", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "handler response" {
			t.Errorf("Expected 'handler response', got %q", w.Body.String())
		}
		if !handlerCalled {
			t.Error("Handler should have been called for allowed request")
		}
	})

	// Test middleware with SaveMatchedRoutePath
	t.Run("middleware with SaveMatchedRoutePath", func(t *testing.T) {
		router := New()
		router.SaveMatchedRoutePath = true

		var middlewareParams Params
		var handlerParams Params

		router.Use(func(next Handle) Handle {
			return func(w http.ResponseWriter, r *http.Request, ps Params) {
				middlewareParams = ps
				next(w, r, ps)
			}
		})

		router.GET("/route/:param", func(w http.ResponseWriter, r *http.Request, ps Params) {
			handlerParams = ps
			w.WriteHeader(http.StatusOK)
		})

		req, _ := http.NewRequest(http.MethodGet, "/route/value", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Middleware runs before SaveMatchedRoutePath is applied, so it won't see the matched route path
		// But the handler should see it correctly
		if middlewareParams.MatchedRoutePath() != "" {
			t.Errorf("Middleware: expected empty matched route (runs before SaveMatchedRoutePath), got %q", middlewareParams.MatchedRoutePath())
		}
		if handlerParams.MatchedRoutePath() != "/route/:param" {
			t.Errorf("Handler: expected matched route '/route/:param', got %q", handlerParams.MatchedRoutePath())
		}
		if handlerParams.ByName("param") != "value" {
			t.Errorf("Expected param 'value', got %q", handlerParams.ByName("param"))
		}
	})
}

func TestRouterMiddlewareWithPanic(t *testing.T) {
	// Test middleware behavior with panic recovery
	router := New()
	middlewareCalled := false
	panicHandlerCalled := false

	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, p interface{}) {
		panicHandlerCalled = true
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("panic recovered"))
	}

	// Middleware that should be called before panic
	router.Use(func(next Handle) Handle {
		return func(w http.ResponseWriter, r *http.Request, ps Params) {
			middlewareCalled = true
			defer func() {
				// This defer should execute even if next() panics
			}()
			next(w, r, ps)
		}
	})

	// Handler that panics
	router.GET("/panic", func(w http.ResponseWriter, r *http.Request, ps Params) {
		panic("test panic")
	})

	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !middlewareCalled {
		t.Error("Middleware should have been called before panic")
	}
	if !panicHandlerCalled {
		t.Error("Panic handler should have been called")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func BenchmarkRouterWithMiddleware(b *testing.B) {
	router := New()

	// Add a few middleware
	router.Use(func(next Handle) Handle {
		return func(w http.ResponseWriter, r *http.Request, ps Params) {
			next(w, r, ps)
		}
	})

	router.Use(func(next Handle) Handle {
		return func(w http.ResponseWriter, r *http.Request, ps Params) {
			next(w, r, ps)
		}
	})

	router.GET("/bench/:param", func(w http.ResponseWriter, r *http.Request, ps Params) {
		// Minimal handler
	})

	w := new(mockResponseWriter)
	r, _ := http.NewRequest(http.MethodGet, "/bench/test", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, r)
	}
}

func BenchmarkRouterWithoutMiddleware(b *testing.B) {
	router := New()

	router.GET("/bench/:param", func(w http.ResponseWriter, r *http.Request, ps Params) {
		// Minimal handler
	})

	w := new(mockResponseWriter)
	r, _ := http.NewRequest(http.MethodGet, "/bench/test", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, r)
	}
}
