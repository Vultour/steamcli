// Package aggregator provides the means to handle multiple profiles at the same time
package aggregator

import (
	"fmt"
	"steamcli/api/profile"
	"steamcli/cache"

	log "github.com/sirupsen/logrus"
)

// Aggregator aggregates multiple Steam profile clients
type Aggregator struct {
	Clients ClientMap
	Cache   *cache.Cache
}

// ClientMap is a map between a user's steam ID and their API Client
type ClientMap map[string]*profile.Client

// New returns an initialized Aggregator struct
func New() *Aggregator {
	a := &Aggregator{
		Clients: make(ClientMap),
		Cache:   cache.New(),
	}
	return a
}

// AddClient initializes and adds a new client to the Aggregator
func (a *Aggregator) AddClient(id string) error {
	var newClient *profile.Client
	var err error

	if _, ok := a.Clients[id]; ok {
		return fmt.Errorf("The client is already present: '%s'", id)
	}

	if p, found := a.Cache.Profiles.Find(id); found {
		log.WithField("name", p.SteamID).Debug("Reusing cached client profile")
		newClient = profile.NewClientPre(p)
	} else {
		log.Debug("Creating new client")
		newClient, err = profile.NewClient(id)
		if err != nil {
			return err
		}
		a.Cache.Profiles.Add(&newClient.Profile)
	}

	log.WithFields(log.Fields{
		"id":   id,
		"name": newClient.Profile.SteamID,
	}).Info("New client:")
	a.Clients[id] = newClient
	return nil
}
