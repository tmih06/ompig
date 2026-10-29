module example.com/benchext10/standalone/bench-ext-06

go 1.26

// Ompig resolves the SDK requirement to the version-matched staged SDK at build time.
require (
	example.com/benchext10 v0.5.3
	github.com/tmih06/ompig/extensions/sdk v0.3.1
)

replace example.com/benchext10 => ../..
