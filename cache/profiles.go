package cache

import (
	"steamcli/objects"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Add adds the specified profile to the cache.
func (p *ProfileCache) Add(profile *objects.XMLProfile) {
	if profileExpired(profile) {
		log.WithFields(log.Fields{
			"name":    profile.SteamID,
			"id":      profile.SteamID64,
			"updated": profile.Updated,
		}).Warning("Attempted to add an expired profile to the cache")
		return
	}

	updated := false
	for i := range *p { // Update all possible copies of the profile (how?!)
		if (*p)[i].SteamID64 == profile.SteamID64 {
			(*p)[i] = profile
			updated = true
		}
	}

	if !updated {
		*p = append(*p, profile)
	}
}

// Find searches the profile cache for the specified ID.
func (p *ProfileCache) Find(id string) (*objects.XMLProfile, bool) {
	log.WithField("id", id).Debug("Searching for profile")
	p.PurgeExpired()
	for _, p := range *p {
		if strconv.FormatInt(p.SteamID64, 10) == id {
			return p, true
		}
		if (p.CustomURL != "") && (strings.ToLower(p.CustomURL) == strings.ToLower(id)) {
			return p, true
		}
	}

	return nil, false
}

// FindGame searches all cached profiles for a specified App ID and returns the name.
// Used to backfill invalid games when the store page no longer exists.
func (p *ProfileCache) FindGame(appid int) (string, bool) {
	for _, c := range *p {
		if g, e := c.Games.Contains(appid); e {
			return g.Name, true
		}
	}
	return "", false
}

// Remove deletes the specified profile from the cache.
// Returns true if the profile was deleted (found), otherwise returns false.
func (p *ProfileCache) Remove(id string) bool {
	removed := false
	for p.removeAll(id) { // Loop to delete duplicates (how?!)
		removed = true
		log.WithField("id", id).Info("Removed profile from cache")
	}
	return removed
}

func (p *ProfileCache) removeAll(id string) bool {
	log.WithField("id", id).Debug("Removing all instances of profile")
	index := -1
	for i, pr := range *p {
		if strconv.FormatInt(pr.SteamID64, 10) == id {
			index = i
			break
		}
		if pr.CustomURL == id {
			index = i
			break
		}
	}
	if index != -1 {
		(*p)[index] = nil
		(*p) = append((*p)[:index], (*p)[index+1:]...)
		return true
	}
	return false
}

// PurgeExpired deletes all expired profiles.
func (p *ProfileCache) PurgeExpired() {
	log.Debug("Purging expired profiles")
	index := 99999
	for index >= 0 {
		index = -1
		for i := range *p {
			if profileExpired((*p)[i]) {
				index = i
				break
			}
		}
		if index != -1 {
			log.WithField("name", (*p)[index].SteamID).Debug("Purging profile")
			(*p)[index] = nil
			(*p) = append((*p)[:index], (*p)[index+1:]...)
		}
	}
}

// profileExpired determines whether the specified profile is too old.
func profileExpired(p *objects.XMLProfile) bool {
	return time.Since(p.Updated) > MaxProfileAge
}
