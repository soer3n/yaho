package mocks

import (
	"net/http"

	"github.com/soer3n/yaho/internal/utils"
	"github.com/stretchr/testify/mock"
)

// HTTPClientMock represents mock struct for http client
type HTTPClientMock struct {
	mock.Mock
	utils.HTTPClientInterface
}

// HTTPResponseMock represents struct for mocking an http response
type HTTPResponseMock struct {
	mock.Mock
	http.ResponseWriter
}
