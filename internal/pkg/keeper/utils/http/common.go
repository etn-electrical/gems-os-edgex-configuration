//
// Copyright (C) 2022 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
)

type ErrorResponse struct {
	Message    string `json:"message,omitempty"`
	StatusCode int    `json:"statusCode"`
}

// Helper method to get the body from the response after making the request
func getBody(resp *http.Response) ([]byte, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return body, errors.New("failed to get the body from the response")
	}
	return body, nil
}

// Helper method to make the request and return the response
func makeRequest(req *http.Request) (*http.Response, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		var netErr *net.OpError
		if errors.As(err, &netErr) {
			return nil, errors.New(fmt.Sprintf("%s cannot be reached, this service is not available.", req.URL.Host))
		} else {
			return nil, errors.New("failed to send a http request")
		}
	}
	if resp == nil {
		return nil, errors.New("the response should not be a nil")
	}
	return resp, nil
}

func createRequest(httpMethod string, baseUrl string, requestPath string, requestParams url.Values) (*http.Request, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	u.Path = requestPath
	if requestParams != nil {
		u.RawQuery = requestParams.Encode()
	}
	req, err := http.NewRequest(httpMethod, u.String(), nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// sendRequest will make a request with raw data to the specified URL.
// It returns the body as a byte array if successful and an error otherwise.
func sendRequest(req *http.Request) ([]byte, ErrorResponse) {
	var errResponse ErrorResponse

	resp, err := makeRequest(req)
	if err != nil {
		return nil, errResponse
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := getBody(resp)
	if err != nil {
		return nil, errResponse
	}

	if resp.StatusCode <= http.StatusMultiStatus {
		return bodyBytes, errResponse
	}

	// Handle error response
	e := json.Unmarshal(bodyBytes, &errResponse)
	if e != nil {
		return nil, errResponse
	}

	return nil, errResponse
}

func createRequestWithRawData(httpMethod string, baseUrl string, requestPath string, requestParams url.Values, data interface{}) (*http.Request, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("fail to parse baseUrl, err: %v", err)
	}
	u.Path = requestPath
	if requestParams != nil {
		u.RawQuery = requestParams.Encode()
	}

	jsonEncodedData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode input data to JSON, err: %v", err)
	}

	req, err := http.NewRequest(httpMethod, u.String(), bytes.NewReader(jsonEncodedData))
	if err != nil {
		return nil, fmt.Errorf("failed to create a http request, err: %v", err)
	}
	req.Header.Set(ContentType, ContentTypeJSON)
	return req, nil
}
