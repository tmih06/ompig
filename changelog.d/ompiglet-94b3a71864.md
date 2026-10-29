### Security

- `ompig ompiglet validate` and `ompig ompiglet add` no longer run the lifecycle scripts (`preinstall`, `install`, `postinstall`, `prepare`) of the npm Packages a Ompiglet names, of their dependencies, of the dependencies of its Git Packages, or of the npm or Git package of a Ompiglet source. Before, validating or adding a Ompiglet ran a publisher's `postinstall` before you had looked at it. `ompig ompiglet build` and `ompig --ompiglet` resolve the same way.

### Changed

- `ompig ompiglet validate` and `ompig ompiglet add` accept `--allow-install-scripts`, which installs a Ompiglet's Packages with lifecycle scripts for that one command. Migration: a Ompiglet whose Package needs its install script to produce files must be validated or added with the flag. `ompig install`, `ompig update` and the other Package commands are unchanged and still run lifecycle scripts, as Pi does.
