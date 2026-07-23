# Changelog

## v0.1.5

### Security (K3 re-pass #64)

- Sanitize player login names and chat (say/tell/yell/emote) before cross-client broadcast.
- Reject CRLF/control characters in names; strip injection from player-authored text.

## v0.1.4

### Security (K3 audit #86, grid-hub trust)

- Grid Hub HTTP client: per-world key headers and updated RPC param shapes.
- `claimCharacterLease` after login; `GRID_WORLD_KEY` env support.

## v0.1.2

### Security (K3 audit #56, #57, #39-class)

- Character login requires a secret phrase (bcrypt); legacy sheets migrate on next login.
- Keeper names require `ADMIN_TOKEN` in addition to the name match.
- Fix remote DoS: empty `forgive` no longer panics the server.
- Reject concurrent login when a character name is already connected on this world.

## v0.1.1

Release sync bump (2026-07-21). No functional changes in this tag.

