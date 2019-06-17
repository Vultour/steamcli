// Package cache implements caching structures for various Steam objects
package cache

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/vultour/steamcli/objects"

	log "github.com/sirupsen/logrus"
)

// FileLocation points to the location of the cache file
// There will be an attempt to choose a sensible default if left empty during
// the call to New()
var FileLocation = ""

// The following Max*Age constants define the time after which an item should be
// purged from the cache
const (
	MaxGameAge    = (time.Hour * 24) * 30
	MaxProfileAge = (time.Hour * 12)
)

// Cache implements the steamcli cache
type Cache struct {
	Games    GameCache    `json:"games"`
	Profiles ProfileCache `json:"profiles"`
}

// GameCache contains game objects
type GameCache map[int]*objects.JSONGame

// ProfileCache contains profile objects
type ProfileCache []*objects.XMLProfile

// New returns a newly initialised Cache
func New() *Cache {
	c := &Cache{
		Games:    make(GameCache),
		Profiles: make(ProfileCache, 0, 4),
	}

	if FileLocation == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			log.WithField("err", err).Panic(
				"Cannot determine user's cache location",
			)
		}
		FileLocation = filepath.Join(cacheDir, "steamcli-cache.json")
		log.WithField("file", FileLocation).Debug("Set cache location")
	}
	c.Load()
	c.Profiles.PurgeExpired()
	c.Games.PurgeExpired()

	return c
}

// Save writes the cache to the file pointed at by FileLocation
func (c *Cache) Save() error {
	log.WithFields(log.Fields{
		"size-games": len(c.Games),
	}).Debug("Saving cache")
	gc, err := os.OpenFile(
		FileLocation,
		os.O_RDWR|os.O_TRUNC|os.O_CREATE,
		0644,
	)
	if err != nil {
		log.Debugf("Error: %#v", err)
		return fmt.Errorf("Couldn't open cache file: %s", err)
	}

	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		log.Debugf("Error: %#v", err)
		return fmt.Errorf("Couldn't encode cache: %s", err)
	}

	for len(b) > 0 {
		n, err := gc.Write(b)
		if err != nil {
			log.Debugf("Error: %#v", err)
			return fmt.Errorf("Couldn't write to cache file: %s", err)
		}
		log.WithField("bytes", n).Debug("Wrote data to cache file")
		b = b[n:]
		log.WithField("remaining", len(b)).Debug("Bytes remaining")
	}

	log.Debug("Cache save done")
	return nil
}

// Load reads the cache from the file pointed at by FileLocation
// Panics in case of any error as that's pretty much a complete disaster for the
// system.
func (c *Cache) Load() error {
	if _, err := os.Stat(FileLocation); os.IsNotExist(err) {
		c.Save() // Create the file if it's empty
	}

	gc, err := os.OpenFile(FileLocation, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Debugf("Error: %#v", err)
		panic(fmt.Sprintf("Couldn't open cache file: %s", err))
	}

	var b []byte
	b, err = ioutil.ReadAll(gc)
	if err != nil {
		log.Debugf("Error: %#v", err)
		panic(fmt.Sprintf("Couldn't read cache file: %s", err))
	}

	err = json.Unmarshal(b, c)
	if err != nil {
		log.Debugf("Error: %#v", err)
		panic(fmt.Sprintf("Couldn't decode cache file: %s", err))
	}

	log.WithFields(log.Fields{
		"games": len(c.Games),
	}).Info("Loaded cache")

	return nil
}
