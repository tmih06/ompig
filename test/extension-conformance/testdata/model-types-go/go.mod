module example.com/model-types-go

go 1.26.0

require (
	github.com/tmih06/ompig v0.3.1
	github.com/tmih06/ompig/extensions/sdk v0.3.1
)

replace github.com/tmih06/ompig => ../../../..

replace github.com/tmih06/ompig/extensions/sdk => ../../../../extensions/sdk
