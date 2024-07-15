//
// Copyright (C) 2022 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package dtos

type KV struct {
	Key   string      `json:"key,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type KeyOnly string

// MultiKVResponse defines the Get Key API response dto with keys query param is false
type MultiKVResponse struct {
	KVs []KV `json:"response"`
}

// MultiKeyResponse defines the Get Key response dto with keys query param is true
type MultiKeyResponse struct {
	Keys []KeyOnly `json:"response"`
}
