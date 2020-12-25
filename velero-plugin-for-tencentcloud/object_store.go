/*
Copyright 2017, 2019 the Velero contributors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tencentyun/cos-go-sdk-v5"
	veleroplugin "github.com/vmware-tanzu/velero/pkg/plugin/framework"
)

type cosObjectGetter interface {
	getCosObjectService(bucket string) cosObject
}

// cosObject interface for objectService
type cosObject interface {
	Put(ctx context.Context, name string, r io.Reader, opt *cos.ObjectPutOptions) (*cos.Response, error)
	Head(ctx context.Context, name string, opt *cos.ObjectHeadOptions, id ...string) (*cos.Response, error)
	Get(ctx context.Context, name string, opt *cos.ObjectGetOptions, id ...string) (*cos.Response, error)
	Delete(ctx context.Context, name string, opt ...*cos.ObjectDeleteOptions) (*cos.Response, error)
	GetPresignedURL(ctx context.Context, httpMethod, name, ak, sk string, expired time.Duration, opt interface{}) (*url.URL, error)
}

// cosBucketGetter interface for bucketService
type cosBucketGetter interface {
	getCosBucketService(bucket string) cosBucket
}

// cosBucket
type cosBucket interface {
	Get(ctx context.Context, opt *cos.BucketGetOptions) (*cos.BucketGetResult, *cos.Response, error)
}

// cosCommon
type cosCommon struct {
	insecureSkipTLSVerify bool
	httpClient            http.Client
	region                string
}

// objectServiceGetter
type objectServiceGetter struct {
	cosObjectService *cos.ObjectService
	cosCommon
}

// getCosObjectService return objectService
func (o *objectServiceGetter) getCosObjectService(bucket string) cosObject {

	bucketURL := &cos.BaseURL{
		BucketURL: cos.NewBucketURL(bucket, o.region, o.insecureSkipTLSVerify),
	}

	o.cosObjectService = cos.NewClient(bucketURL, &o.httpClient).Object
	return o.cosObjectService
}

// bucketServiceGetter
type bucketServiceGetter struct {
	cosBucketService *cos.BucketService
	cosCommon
}

// getCosBucketService return bucketService
func (b *bucketServiceGetter) getCosBucketService(bucket string) cosBucket {
	bucketURL := &cos.BaseURL{
		BucketURL: cos.NewBucketURL(bucket, b.region, b.insecureSkipTLSVerify),
	}
	b.cosBucketService = cos.NewClient(bucketURL, &b.httpClient).Bucket
	return b.cosBucketService
}

// ObjectStore
type ObjectStore struct {
	log logrus.FieldLogger

	cosObjectService cosObjectGetter
	cosBucketService cosBucketGetter
}

// newObjectStore
func newObjectStore(logger logrus.FieldLogger) *ObjectStore {
	return &ObjectStore{log: logger}
}

// Init initialization config
func (o *ObjectStore) Init(config map[string]string) error {

	if err := veleroplugin.ValidateObjectStoreConfigKeys(config,
		regionKey,
		insecureSkipTLSVerifyKey,
	); err != nil {
		return err
	}
	var (
		objectService objectServiceGetter
		bucketService bucketServiceGetter
		region        string
	)

	if region = config[regionKey]; region == "" {
		return errors.New("region is empty")
	}
	objectService.region = region
	bucketService.region = region

	if insecureSkipTLSVerifyVal := config[insecureSkipTLSVerifyKey]; insecureSkipTLSVerifyVal != "" {
		var err error
		if objectService.insecureSkipTLSVerify, err = strconv.ParseBool(insecureSkipTLSVerifyVal); err != nil {
			return errors.Wrapf(err, "could not parse %s (expected bool)", insecureSkipTLSVerifyKey)
		}
	} else {
		objectService.insecureSkipTLSVerify = true
		bucketService.insecureSkipTLSVerify = true
	}
	if err := loadEnv(); err != nil {
		return err
	}
	secretId := os.Getenv("TENCENT_CLOUD_SECRETID")
	secretKey := os.Getenv("TENCENT_CLOUD_SECRETKEY")
	o.log.Printf("tencentcloud ak: %s, sk:%s ", secretId, secretKey)
	httpClient := http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretId,
			SecretKey: secretKey,
		},
	}
	objectService.httpClient = httpClient
	bucketService.httpClient = httpClient
	o.cosObjectService = &objectService
	o.cosBucketService = &bucketService
	return nil
}

// PutObject put one object to cos
func (o *ObjectStore) PutObject(bucket, key string, body io.Reader) error {

	objectService := o.cosObjectService.getCosObjectService(bucket)
	_, err := objectService.Put(context.Background(), key, body, nil)
	if err != nil {
		o.log.Infof("PutObject error: ", err.Error())
	}

	return errors.Wrapf(err, "error putting object %s", key)

}

// ObjectExists
func (o *ObjectStore) ObjectExists(bucket, key string) (bool, error) {

	objectService := o.cosObjectService.getCosObjectService(bucket)
	response, err := objectService.Head(context.Background(), key, nil)
	if response != nil {
		if response.StatusCode >= 500 {
			o.log.Printf("End ObjectExists: return: %v, %v", false, err)
			return false, errors.WithStack(err)

		}
		if response.StatusCode >= 400 && response.StatusCode < 500 {
			o.log.Printf("End ObjectExists: return: %v, %v", false, errors.New("Object Not Exists"))
			return false, nil
		}
	}

	return true, nil
}

// GetObject download one object from cos
func (o *ObjectStore) GetObject(bucket, key string) (io.ReadCloser, error) {

	objectService := o.cosObjectService.getCosObjectService(bucket)
	object, err := objectService.Get(context.Background(), key, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	content := object.Body
	return content, nil
}

// ListCommonPrefixes
func (o *ObjectStore) ListCommonPrefixes(bucket, prefix, delimiter string) ([]string, error) {

	bucketService := o.cosBucketService.getCosBucketService(bucket)
	var res []string
	opt := &cos.BucketGetOptions{
		Prefix:    prefix,
		Delimiter: delimiter,
	}

	bucketResponse, _, err := bucketService.Get(context.Background(), opt)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for _, c := range bucketResponse.Contents {
		res = append(res, c.Key)
	}
	return res, nil
}

// ListObjects list objects
func (o *ObjectStore) ListObjects(bucket, prefix string) ([]string, error) {
	bucketService := o.cosBucketService.getCosBucketService(bucket)

	var res []string
	opt := &cos.BucketGetOptions{
		Prefix: prefix,
	}

	bucketResponse, _, err := bucketService.Get(context.Background(), opt)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for _, c := range bucketResponse.Contents {
		res = append(res, c.Key)
	}
	return res, nil
}

// DeleteObject delete object
func (o *ObjectStore) DeleteObject(bucket, key string) error {

	objectService := o.cosObjectService.getCosObjectService(bucket)
	resp, err := objectService.Delete(context.Background(), key, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 {
		return errors.New("delete fail")

	}
	return nil
}

// CreateSignedURL
func (o *ObjectStore) CreateSignedURL(bucket, key string, ttl time.Duration) (string, error) {

	objectService := o.cosObjectService.getCosObjectService(bucket)
	req, err := objectService.GetPresignedURL(context.Background(), http.MethodGet, key,
		os.Getenv("TENCENT_CLOUD_SECRETID"), os.Getenv("TENCENT_CLOUD_SECRETKEY"), ttl, nil)

	if err != nil {
		return "", err
	}
	return req.String(), nil
}
