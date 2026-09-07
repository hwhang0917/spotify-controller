# Contributing to vibe-music

Thanks for helping. Bug reports, feature ideas and pull requests are welcome.

## Before you start

- Check the [issues](https://github.com/hwhang0917/vibe-music/issues) so work
  is not duplicated. For anything bigger than a fix, open an issue first and
  describe the change; it saves both of us time if the approach is agreed on.
- Set up the toolchain as described in [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

## Making a change

1. Fork and branch from `master`.
2. Keep the change small and focused. Match the surrounding code: the same
   naming, comment density and idioms. Comment the *why* when it cannot live
   in the code; the *what* should be obvious from it.
3. No hardcoded ports, paths, URLs or timeouts: use the existing constants or
   config.
4. Both UIs are bilingual. Every new string goes into `i18n.ts` in **English
   and Korean**; keep sentences on their own lines (`\n`) so they do not break
   mid-sentence.
5. Adding a dependency? Verify the package and exact version exist in its
   registry, prefer the latest stable release, and add it to
   `ui/attributions.ts` if it ships in a UI.
6. Never log or store a secret. Keys and tokens are sealed at rest; audit log
   lines say that a key was set, never what it was.
7. Add or extend a test for non-trivial logic (Go tests live next to the
   code; the UIs are gated by `vue-tsc`).
8. Run `make lint` and `make test` before pushing. CI runs the same plus a
   Windows build.

## Commit messages

[Conventional Commits](https://www.conventionalcommits.org/):
`type(scope): message`, for example `fix(youtube): page the chart to 50`
or `feat(web): favorites tab`. Types: feat, fix, chore, docs, refactor, test,
style, perf, ci.

## Pull requests

- One topic per PR, against `master`.
- Say what changed and why, and how you tested it (which source, which OS).
  Screenshots for UI changes help.
- Keep Spotify policy in mind: Spotify content is never mixed with other
  sources, tracks keep the Spotify mark and link, and nothing streams audio
  from Spotify or YouTube outside their official players.

## Reporting bugs

Use the bug report template. The admin window's **Info** footer link shows
where `vibe-music.log` lives; the lines around the problem (with names and IPs
removed) are the most useful thing you can attach.
