# vibe-music

Office jukebox with votes. One host PC plays the music; everyone on the same
network opens a web page to see what's playing, request songs, upvote the
queue, and vote to skip. Music sources are plugins: **Local files**,
**Spotify**, and **YouTube Music**. The host's credentials never leave the
host process.

## How it works

```
                 ┌──────────────────────────────┐
 guests ─votes──▶│  vibe-music (one Go binary)  │
 (browser)       │  admin window   : Wails      │
                 │  guest API + UI : Chi + SSE  │
                 │  core           : queue,     │
                 │                   votes, poll│
                 │  sources        : local │ spotify
                 └──────────┬──────────┬────────┘
                            ▼          ▼
                      speaker      Spotify desktop
                    (beep/oto)     client (Web API)
```

- **Core owns the queue.** Sorted by votes, then request time. The same song
  can be requested more than once. Skip happens when a configurable share of
  connected guests votes (default 50%). The admin can always skip.
- **Local files and YouTube mix; Spotify plays alone.** Each source has a Use
  switch in the admin. With Local and YouTube on, guests pick a source per
  search and the queue mixes them, the player switching source track by
  track. Spotify's developer policy forbids mixing its content with other
  audio, so turning Spotify on turns the others off (and vice versa), after a
  confirmation whenever songs would be dropped.
- **Spotify is a remote control.** Playback happens in the Spotify desktop
  client on the host. vibe-music only sends Web API player commands, so it is a
  non-streaming app. Guests never hold a token; the server searches and queues
  on their behalf.
- **YouTube is the official embedded player.** Search goes through the YouTube
  Data API v3 with the host's own key; playback runs in YouTube's IFrame player
  inside the admin window, which Go drives through Wails events. No audio is
  fetched or decoded by vibe-music, so it stays within YouTube's terms.
- **Guests are a cookie plus a display name.** Enough for one vote per person
  and "requested by Kim". The admin can disconnect, remove, or block anyone.
- **Invitation-only mode.** Flip the switch and newcomers need a link like
  `http://<host>:5555/join?invitationCode=K7PM-3QXD`. Codes have a TTL and can
  be revoked. Everyone already in the room stays in when you turn it on.
- **Everything survives a relaunch.** Settings, guests, blocks, invitations and
  the queue live in a SQLite file (pure Go driver, no CGO).

## Requirements

- Go 1.25+, Node.js 20+
- [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation):
  `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Windows / macOS:** no C toolchain needed (`CGO_ENABLED=0` builds work).
- **Linux:** Wails needs WebKitGTK and the local player needs ALSA headers
  (`libasound2-dev`), both via cgo. See `wails doctor`.
- For Spotify: a Premium account on the host, the Spotify desktop app open on
  the host PC, and your own app registered at the
  [Spotify dashboard](https://developer.spotify.com/dashboard).

## Setup

### Local files

In the admin window, paste one or more folders (one per line), Save, Rescan.
MP3, WAV, FLAC and OGG Vorbis are indexed; tags and embedded artwork are read
where the format has them. Output goes to the OS default audio device.

### Spotify

1. Create an app at the Spotify dashboard. Redirect URI:
   `http://127.0.0.1:27272/callback` (the admin window shows the exact value;
   the port is configurable there). Spotify rejects `localhost` and rejects a
   loopback URI without a port.
2. Paste the Client ID into the admin window and press **Connect**. A browser
   opens for consent; the token is stored in your user config directory with
   `0600` permissions and refreshed automatically.
3. Open the Spotify desktop app on the host and press play once so it shows up
   as a device. Pick it under **Devices** or leave "whatever is active".

Your dashboard app stays in development mode; only the host authenticates, so
the 5-user limit is never an issue.

### YouTube Music

1. In the Google Cloud console, enable **YouTube Data API v3** and create an
   API key. Under *API restrictions* allow only the YouTube Data API. Leave
   *Application restrictions* at **None**: vibe-music calls the API from the
   host PC, so a "Websites" (HTTP referrer) restriction rejects every request.
2. Paste the key into the admin window's YouTube card and save. It is kept in
   its own `0600` file, not in the database.
3. Switch the source to YouTube. The player appears in the admin's player card
   and must stay visible while YouTube is active; that is a YouTube embed rule.
   If the browser engine refuses autoplay after a relaunch, click the player
   once.

Quota: a search costs 100 units of the default 10,000 per day, so roughly 100
guest searches a day. The guest page debounces typing to make that last.
Search is limited to YouTube's Music category.

## Run

Start the guest server from the admin window and share the URL it shows.
Guests type a name, then search, request, upvote, and vote to skip.

Data lives in `$XDG_CONFIG_HOME/vibe-music/` (Linux),
`~/Library/Application Support/vibe-music/` (macOS), or `%AppData%\vibe-music\`
(Windows). Override the directory with `VIBE_MUSIC_DIR`. It holds
`vibe-music.db` (settings, guests, invitations, queue), `vibe-music.log`
(JSON lines from the app and every guest request; one previous file is kept
once it passes 5 MB) and, once connected, `spotify-token.json`, all `0600`.
The admin window's footer has **Info** (these paths, with copy buttons) and
**Open source licenses**; the guest page footer has the same licenses link (`ui/attributions.ts`, direct dependencies only:
update it when adding one).

### What is and isn't stored

- Guest identity is a random cookie. The database keeps only its SHA-256, so
  reading the file does not let anyone impersonate a guest.
- Invitation codes are stored as SHA-256 too. The plaintext is shown in the
  admin window only for codes created since the app was launched.
- The Spotify token and the YouTube API key must be usable, so they cannot be
  hashed. Each stays in its own `0600` file outside the database. Encrypting it with a key kept next to it
  would add nothing; the OS keychain is the upgrade path if the host is shared.
- bcrypt is not used because there are no passwords: every secret here is a
  high-entropy random token, where a fast hash is the correct choice.

## Develop

```sh
make install     # Go modules, Wails CLI, both frontends
make dev         # admin window with hot reload
make dev-web     # guest UI on :5173, proxies /api to the guest server
make test
make lint        # gofmt + go vet
```

## Build

```sh
make build           # this machine: both UIs embedded, output in build/bin/
make build-windows   # CGO-free Windows .exe from any OS
```

`make help` lists every target. `wails.json` chains the guest UI build into
`frontend:build`, so `wails build` alone also produces the whole binary.

## Layout

```
main.go, app.go            Wails app: bindings, config, guest server lifecycle
logging.go                 JSON log file in the data dir; std log routed into it
internal/source/           Source interface (Track, Playback, ArtworkProvider)
internal/source/local/     folder scan, tags, search, beep playback
internal/source/spotify/   PKCE connect, Web API remote control, end detection
internal/source/youtube/   Data API search, embedded IFrame player control
internal/source/fake/      in-memory source for tests
internal/player/           queue, votes, skip threshold, poll loop, state fan-out
internal/config/           settings (in the store) and the Spotify token file
internal/store/            SQLite: settings, guests, invitations, queue
internal/server/           Chi: guest cookie + name, /api/*, SSE, SPA fallback
ui/theme.css               DESIGN.md tokens mapped onto shadcn-vue's CSS variables
ui/i18n.ts                 framework-free EN/KO lookup shared by both UIs
ui/attributions.ts         open-source list shown by both UIs
frontend/                  admin UI (Vue + Vite + Tailwind + shadcn-vue), embedded by Wails
web/                       guest UI (Vue + Vite + Tailwind + shadcn-vue), embedded via web/embed.go
```

### UI stack

Both apps use [shadcn-vue](https://www.shadcn-vue.com/) components copied into
`src/components/ui/` (reka-ui primitives, Tailwind v4, lucide icons). Add more with

```sh
cd web && npx shadcn-vue@latest add dialog   # or frontend/
```

The theme in `ui/theme.css` maps DESIGN.md's Geist tokens onto shadcn's
variables, so components pick up the ink/hairline look without per-component
overrides. TypeScript is pinned to 5.x in both apps: Vue's SFC compiler needs
the TS 5 JavaScript API to resolve the imported prop types shadcn components
use, and the TypeScript 7 package does not ship it.

### Adding a source

Implement `source.Source` in `internal/source/<name>/`, add it to the
`Sources` list in `app.go`, and give it a card in the admin UI. Nothing else
changes. Implement `source.ArtworkProvider` if artwork is not a public URL.

## Spotify policy notes

- Every Spotify track shown to guests carries the Spotify mark and a
  "Listen on Spotify" link, per Spotify's design guidelines. Check the icon in
  `web/src/SpotifyMark.vue` against the official brand kit before shipping.
- Spotify is licensed for personal, non-commercial use. Where you play it is
  on you.
- Track end is detected by polling once a second, so there is a short gap
  between songs. Repeat mode is switched off on activation because
  repeat-track would defeat end detection.

## License

MIT
