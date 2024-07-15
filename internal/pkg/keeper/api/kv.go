//
// Copyright (C) 2022 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"
	"net/http"
	"net/url"
	"path"

	"github.com/edgexfoundry/go-mod-configuration/v2/internal/pkg/keeper/dtos"
	httpUtils "github.com/edgexfoundry/go-mod-configuration/v2/internal/pkg/keeper/utils/http"
)

// KV is used to manipulate the K/V API
type KV struct {
	c *Caller
}

// KV is used to return a handle to the K/V apis
func (c *Caller) KV() *KV {
	return &KV{c}
}

// Get is used to lookup a single key. The returned pointer
// to the KVPair will be nil if the key does not exist.
func (k *KV) Get(key string) (res dtos.MultiKVResponse, err error) {
	pathParams := url.Values{}
	pathParams.Add(Plaintext, "true")

	url := path.Join(ApiKVRoute, key)
	errResp := httpUtils.GetRequest(&res, k.c.baseUrl, url, pathParams)
	if errResp.StatusCode != 0 {
		return res, errors.New(errResp.Message)
	}
	return res, nil
}

func (k *KV) Keys(key string) (res dtos.MultiKeyResponse, err error) {
	pathParams := url.Values{}
	pathParams.Add(KeyOnly, "true")

	url := path.Join(ApiKVRoute, key)
	errResp := httpUtils.GetRequest(&res, k.c.baseUrl, url, pathParams)
	if errResp.StatusCode == http.StatusNotFound {
		return res, nil
	}
	if errResp.StatusCode != 0 {
		return res, errors.New(errResp.Message)
	}
	return res, nil
}

// Put create/update a single key with value
func (k *KV) Put(key string, data interface{}) error {
	keyPath := path.Join(ApiKVRoute, key)

	value := data
	if byteArray, ok := value.([]byte); ok {
		value = string(byteArray)
	}
	request := dtos.AddKeysRequest{
		Value: value,
	}
	errResp := httpUtils.PutRequest(nil, k.c.baseUrl, keyPath, nil, request)
	if errResp.StatusCode != 0 {
		return errors.New(errResp.Message)
	}
	return nil
}

// PutKeys create/update all keys under a prefix with value
func (k *KV) PutKeys(key string, data interface{}) error {
	keyPath := path.Join(ApiKVRoute, key)
	urlParams := url.Values{}
	urlParams.Add(Flatten, "true")
	value := data
	if byteArray, ok := value.([]byte); ok {
		value = string(byteArray)
	}
	request := dtos.AddKeysRequest{
		Value: value,
	}
	errResp := httpUtils.PutRequest(nil, k.c.baseUrl, keyPath, urlParams, request)
	if errResp.StatusCode != 0 {
		return errors.New(errResp.Message)
	}
	return nil
}

// DeleteKeys delete all keys under a prefix with value
func (k *KV) DeleteKeys(key string) error {
	keyPath := path.Join(ApiKVRoute, key)
	urlParams := url.Values{}
	urlParams.Add(PrefixMatch, "true")

	errResp := httpUtils.DeleteRequest(nil, k.c.baseUrl, keyPath, urlParams)
	if errResp.StatusCode != 0 {
		return errors.New(errResp.Message)
	}
	return nil
}
