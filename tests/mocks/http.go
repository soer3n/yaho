package mocks

import "net/http"

// Get represents mock func for http client get func
func (getter *HTTPClientMock) Get(url string) (*http.Response, error) {
	args := getter.Called(url)
	values := args.Get(0).(*http.Response)
	err := args.Error(1)
	return values, err
}

// Do represents mock func for http client do request func
func (getter *HTTPClientMock) Do(req *http.Request) (*http.Response, error) {
	args := getter.Called(req)
	values := args.Get(0).(*http.Response)
	err := args.Error(1)
	return values, err
}
