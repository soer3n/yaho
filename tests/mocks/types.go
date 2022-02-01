package mocks

import (
	"net/http"

	"github.com/soer3n/yaho/internal/types"
	"github.com/stretchr/testify/mock"
)

// HTTPClientMock represents mock struct for http client
type HTTPClientMock struct {
	mock.Mock
	types.HTTPClientInterface
}

// HTTPResponseMock represents struct for mocking an http response
type HTTPResponseMock struct {
	mock.Mock
	http.ResponseWriter
}
