<p align="center">
   <img src="public/icon.png" width="200px"/>
</p>

# Spotify Controller

> Web based Spotify playlist controller

[![Author](https://img.shields.io/badge/author-RunFridge-green?style=flat)](https://github.com/hwhang0917)
[![License](https://img.shields.io/github/license/RunFridge/film-book)](https://github.com/hwhang0917/spotify-bot/blob/master/LICENSE)

## Translations

- [한국어](docs/README_ko.md)

## Requirements

- Node.js ^20.11.0
- Spotify Premium Account
- GUI Browser

## Setup Guide

1. Clone this repository to your local machine.

2. Install dependencies using `yarn`.

3. Go to your [Spotify Dashboard](https://developer.spotify.com/dashboard/applications) and create a new application for this bot.

4. Add `http://localhost:3000/callback` to Redirect URI.

5. Copy both the Client ID and Client Secret from your newly created Spotify application.

6. Copy `.env.sample` as `.env` to the root directory of this project.

7. Fill in contents on the `.env` as it is guided.

8. Run server using `yarn deploy`.

9. Follow the instructions when browser opens

10. Get your internal private IP address and share `http://${privateIP}:3000` or expose port `3000` to others
