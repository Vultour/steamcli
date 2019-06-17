package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gitlab.com/vultour/steamcli/aggregator"
	"gitlab.com/vultour/steamcli/cache"

	log "github.com/sirupsen/logrus"
)

func main() {
	a := ParseArgs()

	cache.FileLocation = *a.CacheFile
	aggregator.ParallelUpdates = *a.CacheParallel

	if a.Games.Command.Happened() {
		gameCommand(a)
	} else if a.Cache.Command.Happened() {
		cacheCommand(a)
	} else {
		fmt.Print(a.Parser.Usage("No subcommand was specified"))
		os.Exit(4)
	}
}

func gameCommand(a *Arguments) {
	log.WithField("subcmd", ".games").Debug("Subcommand entered")
	agg := aggregator.New()
	for _, v := range *a.IDs {
		log.WithField("id", v).Debug("Adding new ID to aggregator")
		if err := agg.AddClient(v); err != nil {
			log.WithField("err", err).Error("Could not add new client")
		}
	}

	if !*a.NoAutoCache {
		err := agg.UpdateGameCache()
		if err != nil {
			log.WithField("err", err).Error("Could not update game cache")
		}
	}

	if *a.Games.FetchTags {
		err := agg.UpdateGameTags()
		if err != nil {
			log.WithField("err", err).Error("Could not fetch game tags")
		}
	}

	games := agg.Select(*a.Games.Tag, *a.Games.Common, *a.Games.And, *a.Games.Invalid)
	log.WithField("games", len(games)).Debug("Selected games")

	if *a.Games.TagsOnly {
		tags := games.AllTags()
		log.WithField("tags", len(tags)).Debug("Selected all tags")
		for _, t := range tags {
			fmt.Println(t)
		}
		return
	}

	for _, g := range games {
		var name string
		if g.Invalid {
			name = fmt.Sprintf("%s (INVALID)", g.Name)
		} else {
			name = g.Name
		}
		fmt.Printf(
			"%-8s: %-40s : %s\n",
			strconv.Itoa(g.AppID), name, strings.Join(g.CategoriesStrings(), ", "),
		)
	}
}

func cacheCommand(a *Arguments) {
	log.WithField("subcmd", ".cache").Debug("Subcommand entered")
	if a.Cache.Games.Command.Happened() {
		cacheGamesCommand(a)
	}
}

func cacheGamesCommand(a *Arguments) {
	log.WithField("subcmd", ".cache.games").Debug("Subcommand entered")
	if a.Cache.Games.PurgeInvalid.Happened() {
		cacheGamesPurgeInvalid(a)
	} else if a.Cache.Games.PurgeMissingTags.Happened() {
		cacheGamesPurgeMissingTags(a)
	} else if a.Cache.Games.Info.Happened() {
		cacheGamesInfo(a)
	} else if a.Cache.Games.Print.Command.Happened() {
		cacheGamesPrint(a)
	} else if a.Cache.Games.Delete.Command.Happened() {
		cacheGamesDelete(a)
	}
}

func cacheGamesPurgeInvalid(a *Arguments) {
	c := cache.New()
	n := c.Games.PurgeInvalid()
	if err := c.Save(); err != nil {
		log.WithField("err", err).Error("Could not save cache")
	}
	fmt.Printf("Purged %d invalid games from cache\n", n)
}

func cacheGamesPurgeMissingTags(a *Arguments) {
	c := cache.New()
	n := c.Games.PurgeMissingTags()
	if err := c.Save(); err != nil {
		log.WithField("err", err).Error("Could not save cache")
	}
	fmt.Printf("Purged %d games with missing tags from cache\n", n)
}

func cacheGamesInfo(a *Arguments) {
	c := cache.New()

	fmt.Println("=== Game Cache Information ===")
	fmt.Printf("Total games: %d\n", len(c.Games))
	fmt.Printf("Unique tags: %d\n", len(c.Games.AllTags()))
}

func cacheGamesPrint(a *Arguments) {
	c := cache.New()

	games := c.Games.Select(
		*a.Cache.Games.Print.Tag,
		a.Cache.Games.Print.AppIDInt,
		*a.Cache.Games.Print.And,
		*a.Cache.Games.Print.Invalid,
	)
	for _, g := range games {
		fmt.Printf(
			"%-8s: %-32s : %v\n",
			strconv.Itoa(g.AppID), g.Name, strings.Join(g.CategoriesStrings(), ", "),
		)
	}
}

func cacheGamesDelete(a *Arguments) {
	c := cache.New()

	n := 0
	for _, id := range a.Cache.Games.Delete.AppIDInt {
		if !c.Games.Delete(id) {
			log.WithField("id", id).Error("Failed to remove game")
		} else {
			n++
		}
	}

	for _, name := range *a.Cache.Games.Delete.Name {
		if !c.Games.DeleteByName(name) {
			log.WithField("name", name).Error("Failed to remove game")
		} else {
			n++
		}
	}

	fmt.Printf(
		"Removed %d games from cache (requested %d)\n",
		n,
		len(a.Cache.Games.Delete.AppIDInt)+len(*a.Cache.Games.Delete.Name),
	)
}
