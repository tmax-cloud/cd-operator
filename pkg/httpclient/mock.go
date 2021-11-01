package httpclient

import "net/http"

type MockHTTPClient struct {
	DoFunc  func(*http.Request) (*http.Response, error)
	GetFunc func(url string) (*http.Response, error)
}

func (H MockHTTPClient) Do(r *http.Request) (*http.Response, error) {
	return H.DoFunc(r)
}

func (H MockHTTPClient) Get(url string) (*http.Response, error) {
	return H.GetFunc(url)
}
