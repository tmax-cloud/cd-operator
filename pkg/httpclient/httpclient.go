package httpclient

import (
	"net/http"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
	Get(url string) (*http.Response, error)
}
