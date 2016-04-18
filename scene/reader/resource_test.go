package reader

import (
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLocalResource(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	res, err := newResource(thisFile, nil)
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
	res, err := newResource(fetchUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Close()

	fetchUrl = server.URL + "/file-not-found.foo"
	expError := fmt.Sprintf("resource: could not fetch '%s': status %d", fetchUrl, 404)
	_, err = newResource(fetchUrl, nil)
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

	res1, err := newResource(server.URL+"/foo/file1.go", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res1.Close()
	res2, err := newResource("file2.go", res1)
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
	_, err := newResource("gopher://digging.go", nil)
	if err == nil || err.Error() != expError {
		t.Fatalf("expected to get: %s; got %v", expError, err)
	}
}

func TestResourceConnectionRefusedError(t *testing.T) {
	_, err := newResource("http://localhost:12345/foo.go", nil)
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("expected to get 'connection refused error'; got %v", err)
	}
}

func TestRgba8Texture(t *testing.T) {
	imgRes, err := mockImage(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	defer imgRes.Close()
	defer os.Remove(imgRes.Path())

	tex, err := newTexture(imgRes)
	if err != nil {
		t.Fatal(err)
	}

	if tex.width != 1 || tex.height != 1 {
		t.Fatalf("expected tex dims to be 1x1; got %dx%d", tex.width, tex.height)
	}

	if tex.format != Rgba8 {
		t.Fatalf("expected tex format to be %d; got %d", Rgb8, tex.format)
	}

	expLen := 4
	if len(tex.data) != expLen {
		t.Fatalf("expected tex data len to be %d; got %d", expLen, len(tex.data))
	}
}

func TestRgb32Texture(t *testing.T) {
	imgRes, err := mockImage(image.NewRGBA64(image.Rect(0, 0, 1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	defer imgRes.Close()
	defer os.Remove(imgRes.Path())

	tex, err := newTexture(imgRes)
	if err != nil {
		t.Fatal(err)
	}

	if tex.width != 1 || tex.height != 1 {
		t.Fatalf("expected tex dims to be 1x1; got %dx%d", tex.width, tex.height)
	}

	if tex.format != Rgba32 {
		t.Fatalf("expected tex format to be %d; got %d", Rgb32, tex.format)
	}

	expLen := 4 * 4
	if len(tex.data) != expLen {
		t.Fatalf("expected tex data len to be %d; got %d", expLen, len(tex.data))
	}
}

func TestStreamHttpTexture(t *testing.T) {
	serverFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/texture.png" {
			png.Encode(w, image.NewRGBA64(image.Rect(0, 0, 1, 1)))
		} else {
			http.NotFound(w, r)
		}
	})
	server := httptest.NewServer(serverFn)
	defer server.Close()

	imgRes, err := newResource(server.URL+"/texture.png", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer imgRes.Close()

	tex, err := newTexture(imgRes)
	if err != nil {
		t.Fatal(err)
	}

	if tex.width != 1 || tex.height != 1 {
		t.Fatalf("expected tex dims to be 1x1; got %dx%d", tex.width, tex.height)
	}

	if tex.format != Rgba32 {
		t.Fatalf("expected tex format to be %d; got %d", Rgb32, tex.format)
	}

	expLen := 4 * 4
	if len(tex.data) != expLen {
		t.Fatalf("expected tex data len to be %d; got %d", expLen, len(tex.data))
	}
}

func mockResource(payload string) *resource {
	url, _ := url.Parse("embedded")
	return &resource{
		ReadCloser: ioutil.NopCloser(strings.NewReader(payload)),
		url:        url,
	}
}

func mockImage(img image.Image) (res *resource, err error) {
	imgFile := os.TempDir() + "/" + "test.png"
	f, err := os.Create(imgFile)
	if err != nil {
		return nil, err
	}

	err = png.Encode(f, img)
	if err != nil {
		f.Close()
		os.Remove(imgFile)
		return nil, err
	}
	f.Close()

	return newResource(imgFile, nil)
}
