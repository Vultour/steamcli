package profile

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"gitlab.com/vultour/steamcli/api/endpoints"
	"gitlab.com/vultour/steamcli/objects"

	log "github.com/sirupsen/logrus"
)

// Client is a struct used for interacting with the API
type Client struct {
	HTTPClient *http.Client
	baseURL    string
	Profile    objects.XMLProfile
}

// RequestError is returned when a request fails
type RequestError struct {
	Detail     string
	Underlying error
}

var (
	// RequestTimeout is the timeout limit for one request
	RequestTimeout = 15 * time.Second
)

func buildURL(URL string, params map[string]string) string {
	values := make(url.Values)
	values.Add("xml", "1")
	if params != nil {
		for k, v := range params {
			values.Add(k, v)
		}
	}
	return fmt.Sprintf("%s?%s", URL, values.Encode())
}

// NewClient returns a new steamcli Client used for API requests
func NewClient(id string) (*Client, error) {
	client := &Client{
		HTTPClient: &http.Client{
			Timeout: RequestTimeout,
		},
		baseURL: "",
	}

	if _, err := strconv.ParseUint(id, 10, 64); err == nil {
		log.WithField("id", id).Debug("Attempting to use ID as 64bit")
		resp, err := http.Get(
			buildURL(
				fmt.Sprintf("%s/%s/%s", endpoints.Base, endpoints.ID, id),
				nil,
			),
		)
		log.Debugf("Retrieving %s", resp.Request.URL)
		if err != nil {
			return nil, &RequestError{
				Detail:     "Could not perform request",
				Underlying: err,
			}
		}

		profile, err := decodeProfile(&resp.Body)
		if err != nil {
			return nil, &RequestError{
				Detail:     "Could not retrieve profile",
				Underlying: err,
			}
		}
		client.Profile = *profile
		client.baseURL = fmt.Sprintf(
			"%s/%s/%d",
			endpoints.Base,
			endpoints.ID,
			profile.SteamID64,
		)
	} else {
		log.WithField("id", id).Debug("Attempting to use ID as community ID")
		resp, err := http.Get(
			buildURL(
				fmt.Sprintf("%s/%s/%s", endpoints.Base, endpoints.Alias, id),
				nil,
			),
		)
		if err != nil {
			return nil, &RequestError{
				Detail:     "Could not perform request",
				Underlying: err,
			}
		}

		profile, err := decodeProfile(&resp.Body)
		if err != nil {
			return nil, &RequestError{
				Detail:     "Could not retrieve profile",
				Underlying: err,
			}
		}
		client.baseURL = fmt.Sprintf(
			"%s/%s/%d",
			endpoints.Base,
			endpoints.ID,
			profile.SteamID64,
		)
		client.Profile = *profile
	}

	if err := client.GetGames(); err != nil {
		return nil, &RequestError{
			Detail:     "Could not retrieve profile's games",
			Underlying: err,
		}
	}

	return client, nil
}

// NewClientPre creates a client using an existing profile
// This does not perform a web request to fetch the profile unless manually
// triggered.
func NewClientPre(p *objects.XMLProfile) *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout: RequestTimeout,
		},
		baseURL: fmt.Sprintf(
			"%s/%s/%d",
			endpoints.Base, endpoints.ID, p.SteamID64,
		),
		Profile: *p,
	}
}

func (c *Client) get(path string, params map[string]string) (*http.Response, error) {
	url := buildURL(
		fmt.Sprintf("%s/%s", c.baseURL, path),
		params,
	)
	log.WithField("url", url).Debug("Built URL")

	return c.HTTPClient.Get(url)
}

func decodeProfile(stream *io.ReadCloser) (*objects.XMLProfile, error) {
	var profile objects.XMLProfile
	decoder := xml.NewDecoder(*stream)
	err := decoder.Decode(&profile)
	if err != nil {
		return nil, &RequestError{
			Detail:     "Could not decode XML",
			Underlying: err,
		}
	}

	if profile.SteamID64 != 0 {
		profile.Complete()
		log.Debugf("Decoded profile: %#v", profile)
		return &profile, nil
	}

	return nil, &RequestError{Detail: "Could not decode profile"}
}

func (e *RequestError) Error() string {
	return fmt.Sprintf(
		"RequestError: %s | Underlying: %s",
		e.Detail,
		e.Underlying,
	)
}
