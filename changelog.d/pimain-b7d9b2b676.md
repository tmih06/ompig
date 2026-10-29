### Fixed

- `ompig install` and `ompig remove` no longer add ever deeper nested entries to the npm `package-lock.json` when the agent directory is reached through a symlink, such as `~/.pi` linked into a dotfiles repository. Ompig now passes the resolved real path to npm as `--prefix` (or `--cwd` for Bun). <!-- fix: b7d9b2b676 -->
