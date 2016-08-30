package texture

import (
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/achilleasa/go-pathtrace/asset"
)

func TestRgba8Texture(t *testing.T) {
	imgRes, err := mockImage(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	defer imgRes.Close()
	defer os.Remove(imgRes.Path())

	tex, err := New(imgRes)
	if err != nil {
		t.Fatal(err)
	}

	if tex.Width != 1 || tex.Height != 1 {
		t.Fatalf("expected tex dims to be 1x1; got %dx%d", tex.Width, tex.Height)
	}

	if tex.Format != Rgba8 {
		t.Fatalf("expected tex format to be %d; got %d", Rgba8, tex.Format)
	}

	expLen := 4
	if len(tex.Data) != expLen {
		t.Fatalf("expected tex data len to be %d; got %d", expLen, len(tex.Data))
	}
}

func TestRgb32Texture(t *testing.T) {
	imgRes, err := mockImage(image.NewRGBA64(image.Rect(0, 0, 1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	defer imgRes.Close()
	defer os.Remove(imgRes.Path())

	tex, err := New(imgRes)
	if err != nil {
		t.Fatal(err)
	}

	if tex.Width != 1 || tex.Height != 1 {
		t.Fatalf("expected tex dims to be 1x1; got %dx%d", tex.Width, tex.Height)
	}

	if tex.Format != Rgba32F {
		t.Fatalf("expected tex format to be %d; got %d", Rgba32F, tex.Format)
	}

	expLen := 4 * 4
	if len(tex.Data) != expLen {
		t.Fatalf("expected tex data len to be %d; got %d", expLen, len(tex.Data))
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

	imgRes, err := asset.NewResource(server.URL+"/texture.png", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer imgRes.Close()

	tex, err := New(imgRes)
	if err != nil {
		t.Fatal(err)
	}

	if tex.Width != 1 || tex.Height != 1 {
		t.Fatalf("expected tex dims to be 1x1; got %dx%d", tex.Width, tex.Height)
	}

	if tex.Format != Rgba32F {
		t.Fatalf("expected tex format to be %d; got %d", Rgba32F, tex.Format)
	}

	expLen := 4 * 4
	if len(tex.Data) != expLen {
		t.Fatalf("expected tex data len to be %d; got %d", expLen, len(tex.Data))
	}
}

func mockImage(img image.Image) (res *asset.Resource, err error) {
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

	return asset.NewResource(imgFile, nil)
}
