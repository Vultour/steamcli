// Package objects provides complex structures that are used all across steamcli
package objects

import (
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// JSONGame contains the details of a single game
type JSONGame struct {
	Invalid            bool        `json:"_is_invalid"` // Ignore AppID if invalid
	Type               string      `json:"type"`
	Name               string      `json:"name"`
	AppID              int         `json:"steam_appid"`
	RequiredAge        int         `json:"_required_age"`
	RequiredAgeDummy   interface{} `json:"required_age"`
	Description        string      `json:"detailed_description"`
	DescriptionShort   string      `json:"short_description"`
	SupportedLanguages string      `json:"supported_languages"`
	Website            string      `json:"website"`
	Developers         []string    `json:"developers"`
	Publishers         []string    `json:"publishers"`
	Price              struct {
		Currency         string `json:"currency"`
		Initial          int    `json:"initial"`
		Final            int    `json:"final"`
		DiscountPercent  int    `json:"discount_percent"`
		InitialFormatted string `json:"initial_formatted"`
		FinalFormatted   string `json:"final_formatted"`
	} `json:"price_overview"`
	Platforms struct {
		Windows bool `json:"windows"`
		Mac     bool `json:"mac"`
		Linux   bool `json:"linux"`
	} `json:"platforms"`
	Categories []struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
	} `json:"categories"`
	Tags    []string `json:"tags"`
	Updated time.Time
}

// JSONGameList is a slice of games
type JSONGameList []*JSONGame

// Complete computes fields that cannot be determined automatically from JSON
func (g *JSONGame) Complete() {
	// Fuck you Valve, why can't this be one type?
	if x, ok := g.RequiredAgeDummy.(string); ok {
		xx, err := strconv.Atoi(x)
		if err != nil {
			log.Debugf("Couldn't convert RequiredAge to int: %#v", err)
		} else {
			g.RequiredAge = xx
		}
	} else if x, ok := g.RequiredAgeDummy.(int); ok {
		g.RequiredAge = x
	}

	if g.Updated.IsZero() {
		g.Updated = time.Now()
	}
}

// CategoriesStrings returns a slice of all category descriptions
func (g *JSONGame) CategoriesStrings() []string {
	ret := make([]string, 0, len(g.Categories))
	for _, c := range g.Categories {
		ret = append(ret, c.Description)
	}
	return ret
}

// AllTags returns all unique tags across the game slice
func (l *JSONGameList) AllTags() []string {
	set := make(map[string]struct{})
	for _, g := range *l {
		for _, t := range g.Tags {
			set[strings.ToLower(t)] = struct{}{}
		}
	}

	ret := make([]string, 0, len(set))
	for t := range set {
		ret = append(ret, t)
	}
	sort.Strings(ret)
	return ret
}
