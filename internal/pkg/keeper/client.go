//
// Copyright (C) 2022 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package keeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"path"
	"strconv"
	"time"

	"github.com/edgexfoundry/go-mod-configuration/v2/internal/pkg/keeper/api"
	"github.com/edgexfoundry/go-mod-configuration/v2/internal/pkg/keeper/dtos"
	"github.com/edgexfoundry/go-mod-configuration/v2/internal/pkg/keeper/models"
	"github.com/edgexfoundry/go-mod-configuration/v2/internal/pkg/keeper/utils/http"
	"github.com/edgexfoundry/go-mod-configuration/v2/pkg/types"
	"github.com/edgexfoundry/go-mod-messaging/v2/messaging"
	msgTypes "github.com/edgexfoundry/go-mod-messaging/v2/pkg/types"

	"github.com/pelletier/go-toml"
)

const (
	keeperTopicPrefix            = "edgex/configs"
	clientID                     = "ClientId"
	clientIDSuffixRandomInterval = 99999
)

type keeperClient struct {
	keeperUrl      string
	keeperClient   *api.Caller
	configBasePath string
	watchingDone   chan bool
}

// NewKeeperClient creates a new Keeper Client.
func NewKeeperClient(config types.ServiceConfig) *keeperClient {
	client := keeperClient{
		keeperUrl:      config.GetUrl(),
		configBasePath: config.BasePath,
		watchingDone:   make(chan bool, 1),
	}

	client.createKeeperClient(client.keeperUrl)
	return &client
}

func (client *keeperClient) fullPath(name string) string {
	return path.Join(client.configBasePath, name)
}

func (client *keeperClient) createKeeperClient(url string) {
	client.keeperClient = api.NewCaller(url)
}

// IsAlive simply checks if Core Keeper is up and running at the configured URL
func (client *keeperClient) IsAlive() bool {
	err := client.keeperClient.Ping()
	if err != nil {
		return false
	}

	return true
}

// HasConfiguration checks to see if Consul contains the service's configuration.
func (client *keeperClient) HasConfiguration() (bool, error) {
	resp, err := client.keeperClient.KV().Keys(client.configBasePath)
	if err != nil {
		return false, fmt.Errorf("checking configuration existence from Core Keeper failed: %v", err)
	}
	if len(resp.Keys) == 0 {
		return false, nil
	}
	return true, nil
}

func (client *keeperClient) HasSubConfiguration(name string) (bool, error) {
	keyPath := client.fullPath(name)
	resp, err := client.keeperClient.KV().Keys(keyPath)
	if err != nil {
		return false, fmt.Errorf("checking configuration existence from Core Keeper failed: %v", err)
	}
	if len(resp.Keys) == 0 {
		return false, nil
	}
	return true, nil
}

// PutConfigurationToml puts a full toml configuration into Core Keeper
func (client *keeperClient) PutConfigurationToml(configuration *toml.Tree, overwrite bool) error {
	configurationMap := configuration.ToMap()
	err := client.PutConfiguration(configurationMap, overwrite)
	if err != nil {
		return err
	}
	return nil
}

func (client *keeperClient) PutConfiguration(config interface{}, overwrite bool) error {
	var err error
	if overwrite {
		err = client.keeperClient.KV().PutKeys(client.configBasePath, config)
	} else {
		kvPairs := convertMapToKVPairs("", config)
		for _, kv := range kvPairs {
			exists, err := client.ConfigurationValueExists(kv.Key)
			if err != nil {
				return err
			}
			if !exists {
				// Only create the key if not exists in core keeper
				if err = client.PutConfigurationValue(kv.Key, []byte(kv.Value)); err != nil {
					return err
				}
			}
		}
	}
	if err != nil {
		return fmt.Errorf("error occurred while creating/updating configuration, error: %v", err)
	}
	return nil
}

func (client *keeperClient) GetConfiguration(configStruct interface{}) (interface{}, error) {
	exists, err := client.HasConfiguration()
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("the Configuration service (EdgeX Keeper) doesn't contain configuration for %s", client.configBasePath)
	}

	resp, err := client.keeperClient.KV().Get(client.configBasePath)
	if err != nil {
		return nil, err
	}

	err = decode(client.configBasePath+api.KeyDelimiter, resp.KVs, configStruct)
	if err != nil {
		return nil, err
	}
	return configStruct, nil
}

func (client *keeperClient) WatchForChanges(updateChannel chan<- interface{}, errorChannel chan<- error, configuration interface{}, waitKey string) {
	// get the service configuration
	config, err := client.GetConfiguration(&models.ConfigurationStruct{})
	if err != nil {
		errorChannel <- err
		return
	}

	messages := make(chan msgTypes.MessageEnvelope)
	topic := path.Join(keeperTopicPrefix, client.configBasePath, waitKey, "#")
	topics := []msgTypes.TopicChannel{
		{
			Topic:    topic,
			Messages: messages,
		},
	}
	var msgBusConfig models.MessageBusInfo
	configStruct, ok := config.(*models.ConfigurationStruct)
	if !ok {
		configErr := errors.New("configuration data conversion failed")
		close(messages)
		errorChannel <- configErr
		return
	}

	msgBusConfig = configStruct.MessageQueue
	if msgBusConfig.Host == "" || msgBusConfig.Port == 0 || msgBusConfig.Type == "" {
		configErr := errors.New("host, port or type from MessageQueue section is not defined in the configuration")
		close(messages)
		errorChannel <- configErr
		return
	}
	if msgBusConfig.Optional != nil {
		if clientId, ok := msgBusConfig.Optional[clientID]; ok {
			// create unique mqtt client id to prevent missing events during subscription
			randomSuffix := strconv.Itoa(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(clientIDSuffixRandomInterval)) // nolint:gosec
			msgBusConfig.Optional[clientID] = clientId + "-" + randomSuffix
		}
	}

	messageBus, err := messaging.NewMessageClient(msgTypes.MessageBusConfig{
		SubscribeHost: msgTypes.HostInfo{
			Host:     msgBusConfig.Host,
			Port:     msgBusConfig.Port,
			Protocol: msgBusConfig.Protocol,
		},
		Type:     msgBusConfig.Type,
		Optional: msgBusConfig.Optional,
	})
	if err != nil {
		close(messages)
		errorChannel <- err
		return
	}
	// connect to the message bus
	if conErr := messageBus.Connect(); conErr != nil {
		close(messages)
		errorChannel <- conErr
		return
	}
	watchErrors := make(chan error)
	err = messageBus.Subscribe(topics, watchErrors)
	if err != nil {
		_ = messageBus.Disconnect()
		errorChannel <- err
		return
	}

	go func() {
		defer func() {
			_ = messageBus.Disconnect()
		}()

		isFirstUpdate := true

		for {
			select {
			case <-client.watchingDone:
				return
			case e := <-watchErrors:
				errorChannel <- e
			case msgEnvelope := <-messages:
				if isFirstUpdate {
					// send message to channel once the watcher connection is established
					// for go-mod-bootstrap to ignore the first change event
					// refer to https://github.com/edgexfoundry/go-mod-bootstrap/blob/main/bootstrap/config/config.go#L478-L484
					isFirstUpdate = false
					updateChannel <- "watch config change subscription established"
					continue
				}
				if msgEnvelope.ContentType != http.ContentTypeJSON {
					continue
				}
				var respKV dtos.KV
				err := json.Unmarshal(msgEnvelope.Payload, &respKV)
				if err != nil {
					continue
				}
				keyPrefix := path.Join(client.configBasePath, waitKey)
				err = decode(keyPrefix, []dtos.KV{respKV}, configuration)
				updateChannel <- configuration
			}
		}
	}()

	// send empty message to the channel when the watch key change subscription established
	messages <- msgTypes.MessageEnvelope{}
}

func (client *keeperClient) StopWatching() {
	client.watchingDone <- true
}

func (client *keeperClient) ConfigurationValueExists(name string) (bool, error) {
	keyPath := client.fullPath(name)
	res, err := client.keeperClient.KV().Keys(keyPath)
	if err != nil {
		return false, fmt.Errorf("checking configuration existence from Core Keeper failed: %v", err)
	}
	if len(res.Keys) == 0 {
		return false, nil
	}
	return true, nil
}

func (client *keeperClient) GetConfigurationValue(name string) ([]byte, error) {
	keyPath := client.fullPath(name)
	resp, err := client.keeperClient.KV().Get(keyPath)
	if err != nil {
		return nil, err
	}
	if len(resp.KVs) == 0 {
		return nil, fmt.Errorf("%s configuration not found", name)
	}

	var valueStr string
	switch value := resp.KVs[0].Value.(type) {
	case string:
		valueStr = value
	case int:
		valueStr = strconv.Itoa(value)
	case int8:
		value8 := int(value)
		valueStr = strconv.Itoa(value8)
	case int16:
		value16 := int(value)
		valueStr = strconv.Itoa(value16)
	case int32:
		value32 := int(value)
		valueStr = strconv.Itoa(value32)
	case int64:
		value64 := int(value)
		valueStr = strconv.Itoa(value64)
	case float32:
		valueF64 := float64(value)
		valueStr = strconv.FormatFloat(valueF64, 'g', -1, 32)
	case float64:
		valueStr = strconv.FormatFloat(value, 'g', -1, 64)
	case bool:
		valueStr = strconv.FormatBool(value)
	case nil:
		valueStr = ""
	default:
		valueStr = fmt.Sprintf("%v", value)
	}

	return []byte(valueStr), nil
}

func (client *keeperClient) PutConfigurationValue(name string, value []byte) error {
	keyPath := client.fullPath(name)
	err := client.keeperClient.KV().Put(keyPath, value)
	if err != nil {
		return fmt.Errorf("unable to JSON marshal configStruct, err: %v", err)
	}
	return nil
}
