package aggregator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.com/vultour/steamcli/objects"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type gameMatcher []*map[int]struct{}

// GameEndpoint is the Steam Store's endpoint for one or more games
const GameEndpoint = "https://store.steampowered.com/api/appdetails/?appids="

// GameEndpointHTML is the human-readable store endpoint for a single game
// The 'API' endpoint does not return tags, so we need to parse the HTML version
const GameEndpointHTML = "https://store.steampowered.com/app/"

// ParallelUpdates defines how many games will be fetched at one time from store
var ParallelUpdates = 1

// Select returns games across all profiles matching the specified criteria
func (a *Aggregator) Select(tags []string, common, and, invalid bool) objects.JSONGameList {
	matcher := make(gameMatcher, 0, 3)
	for _, c := range a.Clients {
		games := make(map[int]struct{})
		for _, g := range c.Profile.Games {
			log.WithField("id", g.AppID).Debug("Adding game to matcher")
			games[g.AppID] = struct{}{}
		}
		matcher = append(matcher, &games)
		log.WithField("size", len(games)).Debug("Created matcher section")
	}
	var wantedIDs []int
	if common {
		wantedIDs = matcher.Common()
	} else {
		wantedIDs = matcher.All()
	}

	return a.Cache.Games.Select(tags, wantedIDs, and, invalid)
}

// UpdateGameCache updates the aggregator game cache.
// This fetches the details of every game owned across all clients and stores
// it in the cache.
func (a *Aggregator) UpdateGameCache() error {
	gameIDs := make(map[int]struct{})
	for _, c := range a.Clients {
		for id := range c.Profile.Games {
			if _, cached := a.Cache.Games.Get(id); !cached {
				gameIDs[id] = struct{}{}
			}
		}
	}
	log.WithField("n", len(gameIDs)).Debug("Accumulated game IDs")

	c := http.Client{Timeout: time.Second * 10}
	for len(gameIDs) > 0 {
		nextIDs := make([]string, 0, ParallelUpdates)
		i := 0
		for g := range gameIDs {
			nextIDs = append(nextIDs, strconv.Itoa(g))
			i++
			if i >= ParallelUpdates {
				break
			}
		}
		log.WithField("ids", strings.Join(nextIDs, ",")).Debug("Fetching games")

		url := fmt.Sprintf("%s%s", GameEndpoint, strings.Join(nextIDs, ","))
		log.Debugf("Built URL: %s", url)
		r, err := c.Get(url)
		if err != nil {
			return fmt.Errorf("could not retrieve data from the store: %s", err)
		}
		defer r.Body.Close()

		log.WithField("status", r.StatusCode).Debug("Got response")

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("couldn't read response body: %s", err)
		}

		if strings.TrimSpace(string(b)) == "null" {
			return fmt.Errorf("rate limit exceeded or unsupported parallelisation number used")
		}

		// Anonymous struct to deal with the API uglyness
		obj := map[string]*struct {
			Success bool              `json:"success"`
			Data    *objects.JSONGame `json:"data"`
		}{}
		err = json.Unmarshal(b, &obj)
		if err != nil {
			return fmt.Errorf("couldn't decode json: %s", err)
		}

		for _, v := range nextIDs {
			vi, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("couldn't convert AppID: %s", err)
			}

			if vg, exists := obj[v]; exists {
				if !obj[v].Success {
					// Set as invalid and backfill from profile
					log.WithField("id", v).Warning("received invalid response from store")
					pGame, ex := a.Cache.Profiles.FindGame(vi)
					if !ex {
						log.WithField("id", vi).Error("Could not backfill game from profile")
					}
					log.WithField("name", pGame).Debug("Backfilling game name")
					obj[v].Data = &objects.JSONGame{
						AppID:   vi,
						Invalid: true,
						Name:    pGame,
					}
				}

				// Add game to cache
				log.WithFields(log.Fields{
					"id_s": vi,
					"id_i": vg.Data.AppID,
				}).Debug("Adding game to cache")
				vg.Data.Complete()
				a.Cache.Games.Add(vg.Data.AppID, vg.Data)

				// Add a duplicate entry if received mismatch to avoid loop
				if obj[v].Data.AppID != vi {
					log.WithFields(log.Fields{
						"id_requested": vi,
						"id_received":  obj[v].Data.AppID,
					}).Warn("AppID mismatch")
					a.Cache.Games.Add(vi, vg.Data)
					delete(gameIDs, vi)
				}

				delete(gameIDs, vg.Data.AppID)
			} else {
				log.WithFields(log.Fields{
					"id":       v,
					"response": string(b),
				}).Panic("Didn't find game in response")
			}
		}
		if err := a.Cache.Save(); err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 600)
	}

	if err := a.Cache.Save(); err != nil {
		return err
	}
	return nil
}

// UpdateGameTags fetches Game tags for all games that are eligible
func (a *Aggregator) UpdateGameTags() error {
	log.Debug("Updating tags")
	c := &http.Client{Timeout: time.Second * 10}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("could not create cookiejar: %s", err)
	}

	c.Jar = jar
	storeURL, err := url.Parse("https://store.steampowered.com")
	if err != nil {
		return fmt.Errorf("Could not parse store URL: %s", err)
	}
	c.Jar.SetCookies(
		storeURL,
		[]*http.Cookie{
			&http.Cookie{
				Name:    "birthtime",
				Expires: time.Now().Add(time.Hour * 12),
				Domain:  "store.steampowered.com",
				Path:    "/",
				Value:   "156729601",
			},
			&http.Cookie{
				Name:    "lastagecheckage",
				Expires: time.Now().Add(time.Hour * 12),
				Domain:  "store.steampowered.com",
				Path:    "/",
				Value:   "1-0-1987",
			},
			&http.Cookie{
				Name:    "wants_mature_content",
				Expires: time.Now().Add(time.Hour * 12),
				Domain:  "store.steampowered.com",
				Path:    "/",
				Value:   "1",
			},
		},
	)

	for i, g := range a.Cache.Games {
		if g.Tags == nil {
			t, err := fetchTags(c, i)
			if err != nil {
				log.WithFields(log.Fields{
					"err": err,
					"id":  i,
				}).Error("Failed fetching tags")
			}
			log.WithFields(log.Fields{
				"id":   i,
				"tags": t,
			}).Debug("Retrieved tags")
			g.Tags = t
			if err := a.Cache.Save(); err != nil {
				return err
			}
			time.Sleep(time.Second)
		}
	}
	return nil
}

func fetchTags(httpClient *http.Client, appid int) ([]string, error) {
	log.WithField("appid", appid).Debug("Retrieving tags")
	u := fmt.Sprintf("%s%d", GameEndpointHTML, appid)
	log.Debugf("Built URL: %s", u)
	r, err := httpClient.Get(u)
	if err != nil {
		return []string{}, fmt.Errorf("could not retrieve data from the store: %s", err)
	}
	defer r.Body.Close()

	log.WithField("status", r.StatusCode).Debug("Got response")

	doc, err := html.Parse(r.Body)
	if err != nil {
		return []string{}, fmt.Errorf("could not parse HTML: %s", err)
	}

	if !tagReturnSuccess(doc, appid) {
		var s strings.Builder
		err := html.Render(&s, doc)
		if err != nil {
			log.WithField("err", err).Error("Could not render page back into HTML")
		}
		log.Debugf("Validity check failed, content: %s", s.String())
		return []string{}, fmt.Errorf("returned page did not pass validity check")
	}

	tags := findTags(doc)
	if len(tags) < 1 {
		var s strings.Builder
		err := html.Render(&s, doc)
		if err != nil {
			log.WithField("err", err).Error("Could not render page back into HTML")
		}
		log.Debugf("Content: %s", s.String())
		uuu, _ := url.Parse("https://store.steampowered.com")
		for _, c := range httpClient.Jar.Cookies(uuu) {
			log.Debugf("Cookie: %#v", c)
		}
		log.Debugf("Response Headers: %#v", r.Cookies())
		log.WithField("id", appid).Warning("Did not find any tags")
	}
	return tags, nil
}

func tagReturnSuccess(root *html.Node, appid int) bool {
	if root == nil {
		return false
	}
	for e := root; e != nil; e = e.NextSibling {
		if e.DataAtom == atom.Meta {
			for _, a := range e.Attr {
				if (a.Key == "content") && (strings.HasPrefix(a.Val, fmt.Sprintf("https://store.steampowered.com/app/%d", appid))) {
					return true
				}
			}
		}
		if tagReturnSuccess(e.FirstChild, appid) {
			return true
		}
	}
	return false
}

func findTags(root *html.Node) []string {
	ret := make([]string, 0, 8)
	if root == nil {
		return ret
	}
	for e := root; e != nil; e = e.NextSibling {
		if e.DataAtom == atom.A {
			for _, a := range e.Attr {
				if (a.Key == "class") && (a.Val == "app_tag") {
					if (e.FirstChild != nil) && (e.FirstChild.Type == html.TextNode) {
						tag := strings.TrimSpace(e.FirstChild.Data)
						log.Debugf("Found tag: %#v", tag)
						ret = append(ret, tag)
					}
				}
			}
		}
		ret = append(ret, findTags(e.FirstChild)...)
	}
	return ret
}

func (m *gameMatcher) All() []int {
	games := make(map[int]struct{})
	for _, section := range *m {
		for game := range *section {
			games[game] = struct{}{}
		}
	}

	ret := make([]int, 0, 8)
	for game := range games {
		ret = append(ret, game)
	}
	return ret
}

func (m *gameMatcher) Common() []int {
	log.WithField("sections", len(*m)).Debug("Computing common games")
	for _, section := range *m {
		log.WithField("size", len(*section)).Debug("Eliminating items in section")
		for game := range *section {
			inAll := true
			for _, sectionAgain := range *m {
				found := false
				for gameAgain := range *sectionAgain {
					if gameAgain == game {
						found = true
					}
				}
				if !found {
					inAll = false
					break
				}
			}
			if !inAll { // Eliminate entries not present in all maps
				delete(*section, game)
			}
		}
	}
	return m.All() // Maps should only contain common entries
}
