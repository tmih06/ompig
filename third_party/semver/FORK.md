# Ompig fork of Masterminds/semver v3.5.0

This directory is `github.com/Masterminds/semver/v3` v3.5.0, copied from the module proxy (MIT, `LICENSE.txt`), and is a package of the root module, `github.com/tmih06/ompig/third_party/semver` (a `replace` directive would make `go install github.com/tmih06/ompig/cmd/ompig@latest` fail).

The package compiled six regular expressions and built a table in `init`, so every process that linked it paid 1,261 allocations and 199 KB at start, whether or not it parsed a version. The fork compiles the expressions on first use.

## Differences from v3.5.0

1. `lazy.go` (new): `lazyRegexp`, a regular expression compiled by `sync.Once` on first use, with the four methods the package calls (`MatchString`, `FindAllString`, `FindStringSubmatch`, `FindAllStringSubmatch`).
2. `version.go`: `versionRegex` and `looseVersionRegex` are `lazyRegexp` values; their `init` is gone, and the `regexp` import with it.
3. `constraints.go`: `constraintRegex`, `constraintRangeRegex`, `findConstraintRegex` and `validConstraintRegex` are `lazyRegexp` values built from the same expressions (the operator pattern is the constant `constraintOpsPattern`); `init` keeps only the `constraintOps` table. The `regexp` import is gone.
4. `go.mod` is not copied: the package has no dependencies. The import path in `internal/codingagent/selfupdate.go` and `coding/ompiglet/extends.go` is the new one.

The package's own tests run unchanged (`go test ./third_party/semver`).

## Updating

Copy the new upstream release over this directory, repeat the differences above and run the package tests. Offer nothing upstream without the owner.
