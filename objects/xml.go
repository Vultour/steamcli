package objects

import (
	"encoding/xml"
	"time"
)

// XMLProfileError contains an error message associated with the request
type XMLProfileError struct {
	Error string `xml:"error"`
}

// XMLProfile contains information about the retrieved steam profile
type XMLProfile struct {
	XMLName           xml.Name `xml:"profile" json:"-"`
	SteamID           string   `xml:"steamID" json:"steamID"`
	SteamID64         int64    `xml:"steamID64" json:"steamID64"`
	CustomURL         string   `xml:"customURL" json:"customURL"`
	Status            string   `xml:"stateMessage" json:"stateMessage"`
	Privacy           string   `xml:"privacyState" json:"privacyState"`
	PrivacyState      int      `xml:"visibilityState" json:"visibilityState"`
	VACStatus         bool     `xml:"vacBanned" json:"vacBanned"`
	TradeBan          string   `xml:"tradeBanState" json:"tradeBanState"`
	IsLimited         bool     `xml:"isLimitedAccount" json:"isLimitedAccount"`
	MemberSinceString string   `xml:"memberSince" json:"memberSince"`
	Location          string   `xml:"location" json:"location"`
	MemberSince       time.Time
	Games             XMLGameMap `json:"games"`
	Updated           time.Time
}

// XMLProfileGame contains information about a user's game activity
type XMLProfileGame struct {
	XMLName          xml.Name `xml:"game" json:"-"`
	Name             string   `xml:"name" json:"name"`
	AppID            int      `xml:"appID" json:"appID"`
	LinkStore        string   `xml:"storeLink" json:"-"`
	LinkStats        string   `xml:"statsLink" json:"-"`
	LinksStatsGlobal string   `xml:"globalStatsLink" json:"-"`
	PlaytimeTwoWeeks string   `xml:"hoursLast2Weeks" json:"-"`
	PlaytimeTotal    string   `xml:"hoursOnRecord" json:"playtime_total"`
}

// XMLGameMap is a map of profile's games
type XMLGameMap map[int]*XMLProfileGame

// XMLProfileGames contains all of the user's game data
type XMLProfileGames struct {
	XMLName xml.Name         `xml:"gamesList" json:"-"`
	Games   []XMLProfileGame `xml:"games>game" json:"games"`
}

// Complete computes additional fields in SteamProfile
func (profile *XMLProfile) Complete() {
	var err error

	if profile.MemberSince.IsZero() {
		profile.MemberSince, err = time.Parse("January 2, 2006", profile.MemberSinceString)
		if err != nil {
			panic(err)
		}
	}

	if profile.Updated.IsZero() {
		profile.Updated = time.Now()
	}

	if profile.Games == nil {
		profile.Games = make(map[int]*XMLProfileGame)
	}
}

// Contains determines whether the specified appid exists in the game list
func (g *XMLGameMap) Contains(appid int) (*XMLProfileGame, bool) {
	gm, ex := (*g)[appid]
	return gm, ex
}

// ContainsByName determines whether the specified game name exists in the game list
func (g *XMLGameMap) ContainsByName(game string) bool {
	for _, gm := range *g {
		if gm.Name == game {
			return true
		}
	}
	return false
}
