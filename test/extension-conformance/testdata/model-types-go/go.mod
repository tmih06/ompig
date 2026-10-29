module example.com/model-types-go

go 1.26.0

require (
	github.com/tmih06/ompig v0.3.1-0.20261026060701-a6e86b8979d5
	github.com/tmih06/ompig/extensions/sdk v0.3.1-0.20261026060701-a6e86b8979d5
)

replace github.com/tmih06/ompig => ../../../..

replace github.com/tmih06/ompig/extensions/sdk => ../../../../extensions/sdk
