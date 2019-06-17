package cache

import (
	"steamcli/objects"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Add inserts the specified game into the cache
func (g *GameCache) Add(appid int, game *objects.JSONGame) {
	(*g)[appid] = game
}

// Get returns the game specified by appid
// The bool returns true if the game was found in the cache, false otherwise
func (g *GameCache) Get(appid int) (game *objects.JSONGame, exists bool) {
	gm, e := (*g)[appid]
	return gm, e
}

// Select returns games matching the specified criteria
func (g *GameCache) Select(tags []string, appids []int, and, invalid bool) objects.JSONGameList {
	log.WithFields(log.Fields{
		"tags":    tags,
		"appids":  appids,
		"and":     and,
		"invalid": invalid,
	}).Debug("Selecting games")

	appIDMap := make(map[int]struct{})
	for _, id := range appids {
		appIDMap[id] = struct{}{}
	}

	ret := make(objects.JSONGameList, 0, 16)
	for _, game := range *g {
		// Matching app ID?
		if len(appIDMap) > 0 {
			if _, appIDMatch := appIDMap[game.AppID]; !appIDMatch {
				log.WithField("id", game.AppID).Debug("Skipping game, appid mismatch")
				continue
			}
		}

		// Invalid?
		if (tags == nil) || (len(tags) < 1) {
			if invalid || (!invalid && !game.Invalid) {
				ret = append(ret, game)
				continue
			}
		}

		// Matching tags?
		done := false
		for _, wantedTag := range tags {
			done = false
			for _, tag := range game.Tags {
				// TODO: Store tags in lowercase instead of this shit
				if strings.ToLower(tag) == strings.ToLower(wantedTag) {
					if invalid || (!invalid && !game.Invalid) {
						if !and { // Add immediately on an OR match
							ret = append(ret, game)
							delete(appIDMap, game.AppID)
						}
						done = true
					}
				}
				if done {
					break
				}
			}
			if (and && !done) || (!and && done) {
				// Break if one OR tag matched, or any AND tag didn't
				break
			}
		}
		if and && done { // Add if all tags matched
			ret = append(ret, game)
			delete(appIDMap, game.AppID)
		}
	}

	for _, g := range ret {
		delete(appIDMap, g.AppID)
	}

	if len(appIDMap) > 0 {
		idSlice := make([]string, 0, len(appIDMap))
		for id := range appIDMap {
			_, e := g.Get(id)
			// Only warn if the app does not exist at all, or if it got skipped
			// even with invalid option on. Don't warn on skipped invalid games
			// with the invalid option off!
			if (!e) || invalid {
				idSlice = append(idSlice, strconv.Itoa(id))
			}
			if !e {
				log.WithField("id", id).Error("Game doesn't exist in cache")
			}
		}
		if len(idSlice) > 0 {
			log.WithField(
				"ids",
				strings.Join(idSlice, ","),
			).Error("Games not found in cache during Select()")
		}
	}
	return ret
}

// Delete removes the specified AppID & relevant game from the cache
// Returns true if the game was found (& removed), otherwise returns false
func (g *GameCache) Delete(appid int) bool {
	log.WithField("id", appid).Debug("Attempting to delete game from cache")
	if _, found := (*g)[appid]; found {
		log.WithField("id", appid).Debug("Deleting game from cache")
		(*g)[appid] = nil
		delete(*g, appid)
		return true
	}
	log.WithField("id", appid).Debug("Game not found in cache")
	return false
}

// DeleteByName removes the specified game from the cache
// Returns true if the game was found (& removed), otherwise returns false
// Names must be an exact match
func (g *GameCache) DeleteByName(game string) bool {
	log.WithField("game", game).Debug("Attempting to delete game from cache by name")
	for id, gm := range *g {
		if gm.Name == game {
			log.WithFields(log.Fields{
				"game":     game,
				"found":    gm.Name,
				"found_id": gm.AppID,
			}).Debug("Found game in cache")
			if g.Delete(id) {
				return true
			}
			log.WithField("game", game).Warn("Failed to remove game from cache")
		}
	}
	return false
}

// PurgeInvalid removes all invalid games from the cache
// Returns number of purged games
func (g *GameCache) PurgeInvalid() int {
	log.Debug("Purging invalid games from cache")
	n := 0
	for id := range *g {
		if (*g)[id].Invalid {
			if g.Delete(id) {
				n++
			} else {
				log.WithField("id", id).Warn("Failed to remove game from cache")
			}
		}
	}
	return n
}

// PurgeMissingTags removes all games with no tags from the cache
// Returns number of purged games
func (g *GameCache) PurgeMissingTags() int {
	log.Debug("Purging games with missing tags from cache")
	n := 0
	for id := range *g {
		if ((*g)[id].Tags == nil) || (len((*g)[id].Tags) < 1) {
			if !g.Delete(id) {
				log.WithField("id", id).Error("Failed to remove game from cache")
			} else {
				n++
			}
		}
	}
	return n
}

// PurgeExpired removes all expired items from the game cache
func (g *GameCache) PurgeExpired() int {
	log.Debug("Purging expired games")
	n := 0
	for id := range *g {
		if gameExpired((*g)[id]) {
			log.WithFields(log.Fields{
				"id":   id,
				"name": (*g)[id].Name,
			}).Debug("Found expired game")
			if !g.Delete(id) {
				log.WithField("id", id).Error("Failed to remove game from cache")
			} else {
				n++
			}
		}
	}
	return n
}

// AllTags returns tags across all games
func (g *GameCache) AllTags() []string {
	tags := make(map[string]struct{})
	for _, gm := range *g {
		for _, t := range gm.Tags {
			tags[t] = struct{}{}
		}
	}

	ret := make([]string, 0, len(tags))
	for t := range tags {
		ret = append(ret, t)
	}
	return ret
}

func gameExpired(g *objects.JSONGame) bool {
	return time.Since(g.Updated) > MaxGameAge
}
