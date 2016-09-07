package asset

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// The Resource class wraps a streamable file or remote Resource.
type Resource struct {
	io.ReadCloser
	url *url.URL
}

// Returns the path to this resource.
func (r *Resource) Path() string {
	return r.url.String()
}

// Return the remote path to this resource. If this is a remote resource then
// this method returns the base path (without leading /) of the remote URL.
// Otherwise, this method returns the same value as Path().
func (r *Resource) RemotePath() string {
	if r.IsRemote() {
		return filepath.Base(r.url.Path)
	}
	return r.Path()
}

// Returns true if the Resource is streamed over http/https.
func (r *Resource) IsRemote() bool {
	return r.url.Scheme != ""
}

// Create a new Resource data stream. If relTo is specified and pathToResource
// does not define a scheme, then the path to the new Resource will be generated
// by concatenating the base path of relTo and pathToResource.
//
// This function can handle http/https URLs by delegating to the net/http package.
// The caller must make sure to close the returned io.ReadCloser to prevent mem leaks.
func NewResource(pathToResource string, relTo *Resource) (*Resource, error) {
	// Replace forward slashes with backslaces and try parsing as a URL
	url, err := url.Parse(strings.Replace(pathToResource, `\`, `/`, -1))
	if err != nil {
		return nil, err
	}

	// If this is a relative url, clone parent url and adjust its path
	if url.Scheme == "" && relTo != nil {
		path := url.Path
		url, _ = url.Parse(relTo.url.String())
		prefix := url.Path
		if url.Scheme == "" {
			prefix, err = filepath.Abs(relTo.url.String())
			if err != nil {
				return nil, fmt.Errorf("resource: could not detect abs path for %s; %s", relTo.url.String(), err.Error())
			}
		}
		url.Path = filepath.Dir(prefix) + "/" + path
	}

	var reader io.ReadCloser
	switch url.Scheme {
	case "":
		reader, err = os.Open(filepath.Clean(url.Path))
		if err != nil {
			return nil, err
		}
	case "http", "https":
		resp, err := http.Get(url.String())
		if err != nil {
			return nil, fmt.Errorf("resource: could not fetch '%s': %s", url.String(), err)
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("resource: could not fetch '%s': status %d", url.String(), resp.StatusCode)
		}
		reader = resp.Body
	default:
		return nil, fmt.Errorf("resource: unsupported scheme '%s'", url.Scheme)
	}

	return &Resource{
		ReadCloser: reader,
		url:        url,
	}, nil
}

// Create a resource from a reader.
func NewResourceFromStream(name string, source io.Reader) *Resource {
	url, _ := url.Parse(name)
	return &Resource{
		ReadCloser: ioutil.NopCloser(source),
		url:        url,
	}
}
