//
// Copyright (C) 2022 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"

	"github.com/edgexfoundry/go-mod-configuration/v2/internal/pkg/keeper/utils/http"
)

type Caller struct {
	baseUrl string
}

// NewCaller creates an instance of Caller
func NewCaller(baseUrl string) *Caller {
	return &Caller{
		baseUrl: baseUrl,
	}
}

func (c *Caller) Ping() error {
	errResp := http.GetRequest(nil, c.baseUrl, ApiPingRoute, nil)
	if errResp.StatusCode != 0 {
		return errors.New(errResp.Message)
	}
	return nil
}
