// Package metadata implements a client for the cloudscale.ch's OpenStack
// metadata API. This API allows a server to inspect information about itself,
// like its server ID.
//
// Documentation for the API is available at:
//
//    https://www.cloudscale.ch/en/api/v1
package cloudscale

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
	"errors"
)

const (
	maxErrMsgLen = 128 // arbitrary max length for error messages

	defaultTimeout = 2 * time.Second
	defaultPath    = "/openstack/2017-02-22/"
)

var (
	defaultMetadataBaseURL = func() *url.URL {
		u, err := url.Parse("http://169.254.169.254")
		if err != nil {
			panic(err)
		}
		return u
	}()
)

// Client to interact with cloudscale.ch's OpenStack metadata API, from inside
// a server.
type MetadataClient struct {
	client  *http.Client
	BaseURL *url.URL
}

// NewClient creates a client for the metadata API.
func NewMetadataClient(httpClient *http.Client) *MetadataClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	client := &MetadataClient{
		client:  &http.Client{Timeout: defaultTimeout},
		BaseURL: defaultMetadataBaseURL,
	}
	return client
}

// Metadata contains the entire contents of a OpenStack's metadata.
// This method is unique because it returns all of the
// metadata at once, instead of individual metadata items.
func (c *MetadataClient) GetMetadata() (*Metadata, error) {
	metadata := new(Metadata)
	err := c.getResource("meta_data.json", func(r io.Reader) error {
		return json.NewDecoder(r).Decode(metadata)
	})
	return metadata, err
}

// ServerID returns the Server's unique identifier. This is
// automatically generated upon Server creation.
func (c *MetadataClient) GetServerID() (string, error) {
	metadata, err := c.GetMetadata()
	if err != nil {
		return "", err
	}
	if metadata.Meta.CloudscaleUUID == "" {
		return "", errors.New("The CloudscaleUUID was not defined in metadata")
	}
	return metadata.Meta.CloudscaleUUID, nil
}

// RawUserData returns the user data that was provided by the user
// during Server creation. User data for cloudscale.ch is a YAML
// Script that is used for cloud-init.
func (c *MetadataClient) GetRawUserData() (string, error) {
	var userdata string
	err := c.getResource("user_data", func(r io.Reader) error {
		userdataraw, err := ioutil.ReadAll(r)
		userdata = string(userdataraw)
		return err
	})
	return userdata, err
}

func (c *MetadataClient) getResource(resource string, decoder func(r io.Reader) error) error {
	url := c.resolve(defaultPath, resource)
	resp, err := c.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.makeError(resp)
	}
	return decoder(resp.Body)
}

func (c *MetadataClient) makeError(resp *http.Response) error {
	body, _ := ioutil.ReadAll(io.LimitReader(resp.Body, maxErrMsgLen))
	if len(body) >= maxErrMsgLen {
		body = append(body[:maxErrMsgLen], []byte("... (elided)")...)
	} else if len(body) == 0 {
		body = []byte(resp.Status)
	}
	return fmt.Errorf("unexpected response from metadata API, status %d: %s",
		resp.StatusCode, string(body))
}

func (c *MetadataClient) resolve(basePath string, resource ...string) string {
	dupe := *c.BaseURL
	dupe.Path = path.Join(append([]string{basePath}, resource...)...)
	return dupe.String()
}
