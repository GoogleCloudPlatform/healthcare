/*
 * Copyright 2019 Google LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package processors

import (
	"context"
	"io/ioutil"

	"google3/third_party/golang/cloud/storage/storage"
)

// TODO: Add unit test for GCSClient. May use gomock.

// GCSClient represents a GCS bucket.
type GCSClient struct {
	c  *storage.Client
	bh *storage.BucketHandle
}

// NewGCSClient returns a GCSClient instance.
func NewGCSClient(bucketName string, c *storage.Client) *GCSClient {
	return &GCSClient{
		c:  c,
		bh: c.Bucket(bucketName),
	}
}

// ObjectHash returns hash of the Object.
func (gb *GCSClient) ObjectHash(ctx context.Context, path string) ([]byte, error) {
	ob := gb.bh.Object(path)
	atr, err := ob.Attrs(ctx)
	if err != nil {
		return nil, err
	}

	return atr.MD5, nil
}

// ObjectContent returns content of the object.
func (gb *GCSClient) ObjectContent(ctx context.Context, path string) ([]byte, error) {
	ob := gb.bh.Object(path)
	reader, err := ob.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return content, nil
}
