# tracker

A Roblox game & player tracker in Go. Watches games for updates (name,
description, or metadata changes) and watches specific Roblox users for
presence changes (online/offline/in-game), posting rich embeds to Discord
webhooks for both.

## Features

- **Game tracking** — poll one or many games at once (single batched Roblox
  API call per interval), alerting on name/description/metadata changes with
  live player count, total visits, and favorites in the embed.
- **Player tracking** — watch specific Roblox users (by username or user
  ID) and get notified when they start/stop playing a game or come
  online/offline, with their avatar and a link to the game.
- **Follow tracking** — get notified when a tracked player follows a new
  account. This is intentionally one-directional: gaining a follower is
  never reported, only who the tracked player themself chooses to follow.
  Un-follows are also silent, by design.
- **Persistent state** — survives restarts without re-alerting on data it
  already knew about; state is written atomically to disk.
- **Resilient networking** — retries with exponential backoff on network
  errors and 5xx responses, and honors `Retry-After` on 429s from both
  Roblox and Discord.
- **Graceful shutdown** — SIGINT/SIGTERM stop both trackers cleanly and
  flush state to disk before exiting.
- Zero external dependencies — pure Go standard library.

## Build (requires make)

```
git clone https://github.com/FloofyIV/tracker
cd tracker
make
```

Or just `go build` for a binary for your current platform.

## Configuration

Everything is configured via environment variables.

| Variable | Required | Default | Description |
|---|---|---|---|
| `PLACES` | for game tracking | — | Semicolon or comma separated Roblox place IDs to watch |
| `WEBHOOK` | if `PLACES` is set | — | Discord webhook URL for game update alerts |
| `ROLE` | no | — | Discord role ID to `@`-ping on game updates |
| `POLL_INTERVAL` | no | `20` | Seconds between game polls |
| `USERS` | for player tracking | — | Semicolon or comma separated Roblox usernames and/or numeric user IDs to watch |
| `PLAYER_WEBHOOK` | no | falls back to `WEBHOOK` | Discord webhook URL for player alerts |
| `PLAYER_ROLE` | no | falls back to `ROLE` | Discord role ID to `@`-ping on player alerts |
| `PLAYER_POLL_INTERVAL` | no | `30` | Seconds between presence polls |
| `FOLLOW_POLL_INTERVAL` | no | `300` | Seconds between checks of who each tracked player follows |
| `STATE_FILE` | no | `state.json` | Path to the persisted state file |
| `HTTP_TIMEOUT` | no | `15` | Per-request HTTP timeout, in seconds |

At least one of `PLACES` or `USERS` must be set. `PLACE` (singular) is still
accepted as an alias for `PLACES` for backwards compatibility.

## Example

```
$ export PLACES="155615604;4924922222"
$ export WEBHOOK="https://discord.com/api/webhooks/xxx/xxx"
$ export USERS="builderman;1"
$ export PLAYER_WEBHOOK="https://discord.com/api/webhooks/yyy/yyy"
$ ./tracker
```

## Testing

```
go test ./...
```

The suite covers config parsing/validation, state persistence, the retry/
backoff HTTP helper, the Discord embed builders, and the change-detection
logic for both game and player/follow tracking (via pure, extracted diff
functions plus a fake in-process Roblox backend) — no real network access
required.

## Docker

```
docker build -t tracker .
docker run -d \
  -e PLACES="155615604" \
  -e WEBHOOK="https://discord.com/api/webhooks/xxx/xxx" \
  -e USERS="builderman" \
  -v tracker-data:/data \
  tracker
```

State is written to `/data/state.json` inside the container (mount a volume
so it survives restarts).
