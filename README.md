# vibe-music

Office jukebox with votes. One host PC plays the music; everyone on the same
network opens a web page to see what's playing and vote on what comes next.
Music sources are plugins. Spotify is one of them, and the host's Spotify
credentials never leave the host process.

> **Status:** v2 rewrite, scaffold only. Nothing plays yet.

## How it works

```
                 ┌──────────────────────────┐
 guests ─votes──▶│  vibe-music (Go binary)  │
 (browser)       │  admin window   : Wails  │
                 │  guest API + UI : Chi    │
                 │  sources        : plugins│
                 └───────────┬──────────────┘
                             ▼
                      host PC speaker
```

- **Admin window** (Wails): configure plugins, users and roles, start/stop the
  guest server, show the join URL.
- **Guest server** (Chi): JSON API under `/api`, embedded Vue guest UI for
  everything else. Guests never talk to Spotify or hold any token; the server
  performs searches and queue changes on their behalf.
- **Sources** are mutually exclusive: one active plugin per session, never a
  mixed queue. This keeps the Spotify plugin a pure remote control for the
  Spotify desktop client, which is what Spotify's developer policy allows.

## Layout

```
main.go, app.go        Wails app; owns the guest http.Server lifecycle
internal/server/       Chi router: /api/* and SPA fallback
web/                   guest UI (Vue + Vite + Tailwind), embedded via web/embed.go
frontend/              admin UI (Vue + Vite + Tailwind), embedded by Wails
build/                 Wails packaging assets
```

## Requirements

- Go 1.25+
- Node.js 20+
- [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation):
  `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Platform deps per `wails doctor`. Windows builds are CGO-free; Linux and
  macOS need the WebView toolchain Wails lists.

## Develop

```sh
wails dev                 # admin window with hot reload
cd web && npm run dev     # guest UI on :5173, proxies /api to the guest server
go test ./...
```

Start the guest server from the admin window, then open the URL it shows on
any device in the same network.

## Build

```sh
wails build               # builds both UIs, embeds them, outputs build/bin/
```

`wails.json` chains the guest UI build into `frontend:build`, so a single
`wails build` produces the whole binary.

## Spotify plugin notes

- Each host registers their own app at the
  [Spotify dashboard](https://developer.spotify.com/dashboard) and pastes the
  Client ID into the admin window. Use the PKCE flow; no client secret.
- Redirect URI must be a loopback IP such as `http://127.0.0.1:PORT/callback`.
  `localhost` is rejected.
- Playback happens in the Spotify desktop client on the host. This app only
  sends Web API player commands.
- Any Spotify metadata shown to guests carries the Spotify logo and a
  "Listen on Spotify" link, per Spotify's design guidelines.
- Spotify is for personal, non-commercial use. Where you play it is on you.

## License

MIT
