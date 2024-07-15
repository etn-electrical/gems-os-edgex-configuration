//
// Copyright (C) 2022 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package keeper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/edgexfoundry/go-mod-configuration/v2/internal/pkg/keeper/api"
	"github.com/edgexfoundry/go-mod-configuration/v2/internal/pkg/keeper/dtos"
	httpUtils "github.com/edgexfoundry/go-mod-configuration/v2/internal/pkg/keeper/utils/http"
)

type MockCoreKeeper struct {
	keyValueStore map[string]dtos.KV
}

func NewMockCoreKeeper() *MockCoreKeeper {
	return &MockCoreKeeper{
		keyValueStore: make(map[string]dtos.KV),
	}
}

func (mock *MockCoreKeeper) Reset() {
	mock.keyValueStore = make(map[string]dtos.KV)
}

func (mock *MockCoreKeeper) Start() *httptest.Server {
	testMockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, api.ApiKVRoute) {
			key := strings.Replace(request.URL.Path, api.ApiKVRoute+"/", "", 1)

			switch request.Method {
			case "PUT":
				body, err := ioutil.ReadAll(request.Body)
				if err != nil {
					log.Printf("error reading request body: %s", err.Error())
				}
				var addKeysRequest dtos.AddKeysRequest
				err = json.Unmarshal(body, &addKeysRequest)
				if err != nil {
					log.Printf("error decode the request body: %s", err.Error())
				}
				query := request.URL.Query()
				_, isFlatten := query[api.Flatten]
				if isFlatten {
					kvPairs := convertMapToKVPairs(key, addKeysRequest.Value)
					for _, kvPair := range kvPairs {
						mock.updateKVStore(kvPair.Key, kvPair.Value)
					}
				} else {
					mock.updateKVStore(key, addKeysRequest.Value)
				}
			case "GET":
				query := request.URL.Query()
				_, allKeysRequested := query[api.KeyOnly]

				var resp interface{}
				pairs, prefixFound := mock.checkForPrefix(key)
				if !prefixFound {
					resp = httpUtils.ErrorResponse{
						Message:    fmt.Sprintf("query key %s not found", key),
						StatusCode: http.StatusNotFound,
					}
					writer.WriteHeader(http.StatusNotFound)
				} else {
					if allKeysRequested {
						var keys []dtos.KeyOnly
						// Just returning array of key paths
						for _, kvPair := range pairs {
							keys = append(keys, dtos.KeyOnly(kvPair.Key))
						}
						resp = dtos.MultiKeyResponse{Keys: keys}
					} else {
						var kvs []dtos.KV

						// Just returning array of key-value pairs
						for _, kvPair := range pairs {
							kv := dtos.KV{
								Key:   kvPair.Key,
								Value: kvPair.Value,
							}
							kvs = append(kvs, kv)
						}
						resp = dtos.MultiKVResponse{KVs: kvs}
					}
					writer.WriteHeader(http.StatusOK)
				}
				writer.Header().Set("Content-Type", "application/json")

				if err := json.NewEncoder(writer).Encode(resp); err != nil {
					log.Printf("error writing data response: %s", err.Error())
				}
			}
		} else if strings.Contains(request.URL.Path, api.ApiPingRoute) {
			switch request.Method {
			case "GET":
				writer.WriteHeader(http.StatusOK)

			}
		}
	}))

	return testMockServer
}

func (mock *MockCoreKeeper) checkForPrefix(prefix string) ([]dtos.KV, bool) {
	var pairs []dtos.KV
	for k, v := range mock.keyValueStore {
		if strings.HasPrefix(k, prefix) {
			pairs = append(pairs, v)
		}
	}
	if len(pairs) == 0 {
		return nil, false
	}
	return pairs, true

}

// updateKVStore updates the value of the specified key from the mock key-value store map
func (mock *MockCoreKeeper) updateKVStore(key string, value interface{}) {
	keyValuePair, found := mock.keyValueStore[key]
	if found {
		keyValuePair.Value = value
	} else {
		keyValuePair = dtos.KV{
			Key:   key,
			Value: value,
		}
	}
	mock.keyValueStore[key] = keyValuePair
}
