package main

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io"
	"net/url"
	"testing"
	"time"
)

type mockObjectServiceGetter struct {
	mock.Mock
}

func (m *mockObjectServiceGetter) getCosObjectService(bucket string) cosObject {
	args := m.Called(bucket)
	return args.Get(0).(cosObject)
}

type mockObjectService struct {
	mock.Mock
}

func (m *mockObjectService) Put(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error) {
	args := m.Called(ctx, name, r, opt)
	return args.Get(0).(*cos.Response), args.Error(1)
}

func (m *mockObjectService) Head(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error) {
	args := m.Called(ctx, name, opt, id)
	return args.Get(0).(*cos.Response), args.Error(1)
}

func (m *mockObjectService) Get(ctx context.Context, name string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error) {
	args := m.Called(ctx, name, opt, id)
	return args.Get(0).(*cos.Response), args.Error(1)
}

func (m *mockObjectService) Delete(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error) {
	args := m.Called(ctx, name, opt)
	return args.Get(0).(*cos.Response), args.Error(1)
}

func (m *mockObjectService) GetPresignedURL(ctx context.Context, httpMethod, name, ak, sk string, expired time.Duration, opt interface{}) (*url.URL, error) {
	args := m.Called(ctx, httpMethod, name, ak, sk, expired, opt)
	return args.Get(0).(*url.URL), args.Error(1)
}

func TestObjectExists(t *testing.T) {
	tests := []struct {
		name           string
		getBlobError   error
		exists         bool
		errorResponse  error
		expectedExists bool
		expectedError  string
	}{
		{
			name:           "getCosObjectService error",
			exists:         false,
			errorResponse:  errors.New("getCosObjectService"),
			expectedExists: false,
			expectedError:  "getBlob",
		},
		{
			name:           "exists",
			exists:         true,
			errorResponse:  nil,
			expectedExists: true,
		},
		{
			name:           "doesn't exist",
			exists:         false,
			errorResponse:  nil,
			expectedExists: false,
		},
		{
			name:           "error checking for existence",
			exists:         false,
			errorResponse:  errors.New("bad"),
			expectedExists: false,
			expectedError:  "bad",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objectServiceGetter := new(mockObjectServiceGetter)
			defer objectServiceGetter.AssertExpectations(t)

			o := &ObjectStore{
				cosObjectService: objectServiceGetter,

			}

			bucket := "b"
			key := "k"

			object := new(mockObjectService)
			defer object.AssertExpectations(t)
			objectServiceGetter.On("getCosObjectService", bucket).Return(object, tc.getBlobError)

			object.On("Exists").Return(tc.exists, tc.errorResponse)

			exists, err := o.ObjectExists(bucket, key)

			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.expectedExists, exists)
		})
	}
}
