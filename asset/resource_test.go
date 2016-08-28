package asset

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLocalResource(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	res, err := NewResource(thisFile, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()
}

func TestHttpResource(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	thisDir := filepath.Dir(thisFile)

	server := httptest.NewServer(http.FileServer(http.Dir(thisDir)))
	defer server.Close()

	fetchUrl := server.URL + "/" + filepath.Base(thisFile)
	res, err := NewResource(fetchUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()

	fetchUrl = server.URL + "/file-not-found.foo"
	expError := fmt.Sprintf("resource: could not fetch '%s': status %d", fetchUrl, 404)
	_, err = NewResource(fetchUrl, nil)
	if err == nil || err.Error() != expError {
		t.Fatalf("expected to get: %s; got %v", expError, err)
	}
}

func TestRelativeResources(t *testing.T) {
	serverHits := 0
	serverFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHits++
		if r.URL.Path == "/foo/file1.go" {
			w.Write([]byte("OK"))
		} else if r.URL.Path == "/foo/file2.go" {
			w.Write([]byte("OK"))
		} else {
			http.NotFound(w, r)
		}
	})
	server := httptest.NewServer(serverFn)
	defer server.Close()

	res1, err := NewResource(server.URL+"/foo/file1.go", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res1.Close()
	res2, err := NewResource("file2.go", res1)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Close()

	if serverHits != 2 {
		t.Fatalf("expected server to receive 2 requests; got %d", serverHits)
	}
}

func TestUnsupportedResourceScheme(t *testing.T) {
	expError := "resource: unsupported scheme 'gopher'"
	_, err := NewResource("gopher://digging.go", nil)
	if err == nil || err.Error() != expError {
		t.Fatalf("expected to get: %s; got %v", expError, err)
	}
}

func TestResourceConnectionRefusedError(t *testing.T) {
	_, err := NewResource("http://localhost:12345/foo.go", nil)
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("expected to get 'connection refused error'; got %v", err)
	}
}
func mockResource(payload string) *Resource {
	url, _ := url.Parse("embedded")
	return &Resource{
		ReadCloser: ioutil.NopCloser(strings.NewReader(payload)),
		url:        url,
	}
}
