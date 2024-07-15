//
// Copyright (C) 2022 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// GetRequest makes the get request and return the body
func GetRequest(returnValuePointer interface{}, baseUrl string, requestPath string, requestParams url.Values) ErrorResponse {
	req, err := createRequest(http.MethodGet, baseUrl, requestPath, requestParams)
	if err != nil {
		return ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    err.Error(),
		}
	}

	res, errResp := sendRequest(req)
	if errResp.StatusCode != 0 {
		return errResp
	}
	// Check the response content length to avoid json unmarshal error
	if returnValuePointer == nil || len(res) == 0 {
		return errResp
	}
	//var keyResp dtos.MultiKeyResponse
	if err := json.Unmarshal(res, &returnValuePointer); err != nil {
		return ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "failed to parse the response body",
		}
	}
	return errResp
}

// PutRequest makes the put JSON request and return the body
func PutRequest(
	returnValuePointer interface{},
	baseUrl string, requestPath string,
	requestParams url.Values,
	data interface{}) ErrorResponse {

	req, err := createRequestWithRawData(http.MethodPut, baseUrl, requestPath, requestParams, data)
	if err != nil {
		return ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    err.Error(),
		}
	}

	res, errResp := sendRequest(req)

	if errResp.StatusCode != 0 {
		return errResp
	}
	// no need to unmarshal the response if returnValuePointer is nil
	if returnValuePointer == nil {
		return ErrorResponse{}
	}
	if err := json.Unmarshal(res, returnValuePointer); err != nil {
		return ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "failed to parse the response body",
		}
	}
	return ErrorResponse{}
}

// DeleteRequest makes the get request and return the body
func DeleteRequest(returnValuePointer interface{}, baseUrl string, requestPath string, requestParams url.Values) ErrorResponse {
	req, err := createRequest(http.MethodDelete, baseUrl, requestPath, requestParams)
	if err != nil {
		return ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    err.Error(),
		}
	}

	res, errResp := sendRequest(req)
	if errResp.StatusCode != 0 {
		return errResp
	}
	// Check the response content length to avoid json unmarshal error
	if returnValuePointer == nil || len(res) == 0 {
		return errResp
	}

	if err := json.Unmarshal(res, &returnValuePointer); err != nil {
		return ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "failed to parse the response body",
		}
	}
	return errResp
}
