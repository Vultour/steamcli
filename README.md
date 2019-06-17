# steamcli
A command line tool and library for combining, filtering, and printing Steam profile data.

## Command Line Tool
### Usage
Explore commands using `steamcli subcommand [subcommand...] --help`

```
usage: steamcli <Command> [-h|--help] [-v|--verbose] [--debug] [--json-log]
                [-i|--id "<value>" [-i|--id "<value>" ...]] [--cache-file
                "<value>"] [--cache-parallel <integer>] [-n|--no-auto-cache]

                Utility for combining, filtering, and printing community
                profile data

Commands:

  games  Interact with the game library
  cache  Manipulate the steamcli cache

Arguments:

  -h  --help            Print help information
  -v  --verbose         Increase logging verbosity
      --debug           Increase logging verbosity even more
      --json-log        Use JSON as logging format
  -i  --id              A steam ID (64bit, STEAM_X:Y:Z, or community ID)
      --cache-file      File to be used for the game cache
      --cache-parallel  How many games to fetch in parallel when getting
                        details. Default: 1
  -n  --no-auto-cache   Don't retrieve details for non-cached games
```

### Notes
- It might take a very long time to run if used on an account with large amount of games, as it fetches about ~1-1.5 games per second. The 'unofficial' steam store API does not allow fetching more than one game at a time anymore.
- Use `--fetch-tags` to also retrieve game tags, this requires requesting and parsing the HTML version as it is not included in the API response.
- Data is cached, games have an expiration of 30 days, profiles 12 hours. See `cache/cache.go`.
- Running with `--fetch-tags` will also retrieve tags for the rest of the games in the cache, not just newly fetched ones.
- `--cache-parallel` can be used to increase the number of games fetched per request from the API. Steam seems to have disabled this functionality so requesting more than one game at a time returns `null`.
- Steam IDs can be specified in three forms: `STEAM_X:Y:Z`, 64bit Steam ID, or a community id (the custom URL nickname, not _any_ name)
- The 'categories' printed after the game name aren't tags, they're the official Steam categories (e.g. "Multi-Player", "Steam Workshop", "In-App Purchases")

### Examples
#### Get all games in an account
```
$ ./steamcli games --id 76561198016990736 | head -n 3
431960  : Wallpaper Engine                         : Steam Achievements, Steam Trading Cards, Steam Workshop, Includes level editor
439700  : Z1 Battle Royale: Test Server            : Multi-player, MMO
504370  : Battlerite                               : Single-player, Multi-player, Online Multi-Player, In-App Purchases, Steam Cloud
```

#### Show tags of all games in the account
Tags will only work with `--fetch-tags` (required at least the first time a game is encountered and cached). Steam doesn't include tags in their 'unofficial' store API call, therefore it needs a second request that parses the full HTML version.
```
$ ./steamcli games --id 76561198016990736 --fetch-tags --tags-only | shuf | head -n 5
illuminati
tactical
blood
mod
turn-based combat
```

#### Show games that match a specific tag
> Hint: You can use the `--tag` parameter more than once to specify more tags. By default games matching _any_ of the tags will be shown, this can be changed so games have to match _all_ tags using `--and`.
```
$ ./steamcli games --id 76561198016990736 --tag arcade
236390  : War Thunder                              : Single-player, MMO, Co-op, Cross-Platform Multiplayer, Steam Achievements, In-App Purchases, Partial Controller Support
11020   : TrackMania Nations Forever               : Single-player, Multi-player, Includes level editor
```


#### Show common games across two accounts
> Hint: All the previous modifiers still work, e.g. `--tag`, `--and`, or `--tags-only`.

> Hint 2: You can see all games across both accounts by dropping `--common`

> Hint 3: `--id` can be used as many times as needed, not just twice
```
$ ./steamcli games --id 76561198016990736 --id 76561198076575909 --common
295110  : Just Survive                             : Multi-player, MMO, Steam Trading Cards
433850  : Z1 Battle Royale                         : Multi-player, Online Multi-Player, In-App Purchases
730     : Counter-Strike: Global Offensive         : Multi-player, Steam Achievements, Full controller support, Steam Trading Cards, Steam Workshop, In-App Purchases, Valve Anti-Cheat enabled, Stats
439700  : Z1 Battle Royale: Test Server            : Multi-player, MMO
362300  : Just Survive Test Server                 : Single-player, Multi-player, MMO
```

### Other Functionality
#### Cache management
Remove cached games that are marked as invalid or don't have any tags associated with them.
```
$ ./steamcli cache games purge-invalid
Purged 11 invalid games from cache

$ ./steamcli cache games purge-missing-tags
Purged 3 games with missing tags from cache
```

#### Cache inspection
```
$ ./steamcli cache games info
=== Game Cache Information ===
Total games: 81
Unique tags: 195
```

`cache print` works just like the main `games` command, but on the whole cache instead of individual accounts.
```
$ ./steamcli cache games print --tag blood
208090  : Loadout                          : Multi-player, Co-op, Steam Achievements, Steam Trading Cards, Partial Controller Support, Steam Cloud, Valve Anti-Cheat enabled
273300  : Outlast: Whistleblower DLC       : Single-player, Downloadable Content, Steam Achievements, Full controller support, Captions available, Steam Cloud
```

## Library "documentation"
[![GoDoc](https://godoc.org/github.com/Vultour/steamcli?status.svg)](https://godoc.org/github.com/Vultour/steamcli)