package profile

import (
	"encoding/xml"
	"io"
	"steamcli/api/endpoints"
	"steamcli/objects"
	"strings"

	log "github.com/sirupsen/logrus"
)

// MaxGamesPerRequest determines how many games are retrieved at once
var MaxGamesPerRequest = 10

// GetProfile retrieves the profile associated with the client
func (c *Client) GetProfile() (*objects.XMLProfile, error) {
	response, err := c.get("", nil)
	if err != nil {
		return &objects.XMLProfile{}, &RequestError{
			Detail:     "Could not perform request",
			Underlying: err,
		}
	}
	profile, err := decodeProfile(&response.Body)
	if err != nil {
		return &objects.XMLProfile{}, &RequestError{
			Detail:     "Could not retrieve profile",
			Underlying: err,
		}
	}
	return profile, nil
}

// GetGames retrieves the user's game activity
func (c *Client) GetGames() error {
	response, err := c.get(endpoints.Games, map[string]string{"tab": "all"})
	if err != nil {
		return &RequestError{
			Detail:     "Could not perform request",
			Underlying: err,
		}
	}
	defer response.Body.Close()

	//if !profileExists(&response.Body) {
	//	return &api.SteamProfileGames{}, &RequestError{
	//		Detail: "The specified profile does not exist",
	//	}
	//}

	var games objects.XMLProfileGames
	decoder := xml.NewDecoder(response.Body)
	err = decoder.Decode(&games)
	if err != nil {
		return &RequestError{
			Detail:     "Could not decode XML",
			Underlying: err,
		}
	}
	log.WithField("games", len(games.Games)).Debug("Retrieved profile games")

	for id := range games.Games {
		log.Debugf("Adding game to profile: %#v", games.Games[id])
		c.Profile.Games[games.Games[id].AppID] = &games.Games[id]
	}

	return nil
}

func profileExists(r *io.ReadCloser) bool {
	var errxml objects.XMLProfileError
	decoder := xml.NewDecoder(*r)
	err := decoder.Decode(&errxml)
	if err == nil {
		if strings.TrimSpace(errxml.Error) == "" {
			return true
		}
	}
	return false
}
