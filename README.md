# vibe-music

Office jukebox with votes. One PC plays the music; everyone on the same Wi-Fi
opens a web page to see what's playing, request songs, upvote the playlist,
and vote to skip. Music can come from **local files**, **Spotify**, or
**YouTube Music**. Your accounts and keys stay on the host PC.

Screens: a small admin window on the host, and a phone-friendly guest page.

## Get it

**Windows:** download `vibe-music.exe` from the
[Releases](https://github.com/hwhang0917/vibe-music/releases) page and run it.
No installer, nothing else to install.

**Linux (amd64) / macOS (Intel):**

```sh
curl -fsSL https://raw.githubusercontent.com/hwhang0917/vibe-music/master/scripts/install.sh | sh
```

Installs to `~/.local/bin/vibe-music` plus a launcher entry (Linux) or
`~/Applications/vibe-music.app` (macOS); set `INSTALL_DIR` to change that,
`VIBE_MUSIC_VERSION` to pin a release. On Linux the app needs WebKitGTK 4.1 at runtime; the script checks
for it and prints the package to install (`libwebkit2gtk-4.1-0` on
Debian/Ubuntu, `webkit2gtk4.1` on Fedora, `webkit2gtk-4.1` on Arch). Apple
Silicon and ARM Linux are not built yet.

Building it yourself, or hacking on it: see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

## Quick start (host)

1. Launch vibe-music. Pick a music source under **Sources** (see below) and
   switch its **Use** toggle on.
2. Press **Start** at the top right. The window shows the join address, e.g.
   `http://192.168.0.12:5555`. Copy it or read it out.
3. Guests open the address on their phone, type a name, and start requesting.

Windows may ask to allow vibe-music through the firewall the first time; say
yes for private networks, or guests cannot reach it.

## Music sources

You can run local files and YouTube together. Spotify plays alone: its rules
forbid mixing its music with other audio, so switching Spotify on switches the
others off (the app asks first if songs would be dropped).

### Local files

Add one or more folders. MP3, WAV, FLAC and OGG are indexed with title,
artist, album, genre, year and length. The first scan of a large library takes
a while (each MP3 has to be read once); after that only new or changed files
are opened. Music plays through the PC's default speakers.

### Spotify

You need a **Spotify Premium** account and a free developer app of your own.
The **(?)** button on the Spotify card walks you through it: create an app at
the Spotify dashboard, add the redirect address the card shows, paste the
Client ID, press **Connect**, approve in the browser. Then open the Spotify
desktop app on this PC and press play once so it shows up as a device.

If the "Web API" checkbox in the dashboard is greyed out, the account you are
logged in with is not Premium.

### YouTube Music

You need a free Google API key. The **(?)** button on the YouTube card walks
you through it: enable *YouTube Data API v3* in the Google Cloud console,
create an API key with **no application restriction** (the app calls Google
from this PC, not from a website), paste it, press **Save key**, then **Test
key**. Playback happens in the small YouTube player inside the admin window,
which has to stay visible while a YouTube song plays.

Google gives 10,000 free units a day. A guest search costs 100, so plan on
about 100 searches a day; the guide explains how to ask for more.

## What guests can do

- See what is playing, who requested it, and the playlist with vote counts.
- **Request a song**: search one source at a time, or browse the YouTube Top
  50, the most-played songs on this host, or their own favorites (hearts, kept
  on their phone only).
- Tap an artist or album to see more from it.
- Upvote songs in the playlist, remove their own requests, and vote to skip.
  A skip happens when half the connected guests vote (the host can change the
  share, and can always skip).
- Requesting a song that is already playing or queued asks first.

## What the host can do

- Pause, resume, skip, seek, set volume, reorder and remove playlist items.
- See connected guests; disconnect, remove, or block anyone.
- **Invitation-only mode**: newcomers need a link from the admin's Invitations
  card. Each link works once and expires; people already in the room stay in.
- Reset the "most played" list.
- Switch the language (English / 한국어). Guests have their own toggle.

Settings, guests, blocks, invitations and the playlist survive a relaunch.

## Privacy and security

- Guests are identified by a random cookie and the name they typed. The host
  sees names, not phones or accounts.
- Your Spotify login token and YouTube key are stored encrypted on the host PC
  (Windows user-account encryption) and never sent to guests.
- The **Info** link in the admin footer shows where the data and the log file
  live. The log records requests and actions on this host only.
- Everything runs on your local network. Nothing is uploaded anywhere except
  the calls to Spotify and Google that you set up.

## Troubleshooting

- **Guests cannot open the address**: same Wi-Fi? Firewall allowed? Try the
  address on the host itself first.
- **"Port already in use"**: change the port on the Server tab and press Start
  again.
- **No network at launch**: Spotify and YouTube stay off with a warning; local
  files still work.
- **Spotify: "Premium required"** or nothing plays: the connected account must
  be Premium and the desktop app must be open and have played once.
- **YouTube: key rejected**: use **Test key**; the message quotes Google's
  reason. Usual causes: the API is not enabled on the key's project, or the key
  has a website restriction. A new or edited key can take a few minutes.
- **The exe shows the wrong icon**: Windows caches icons; rename the file once
  or sign out and in.

## Contributing

Bug reports and ideas go through the
[issue templates](https://github.com/hwhang0917/vibe-music/issues/new/choose);
code changes are described in [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT. Open-source components are listed under **Open source licenses** in both
the admin footer and the guest page footer.
