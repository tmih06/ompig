// This directory holds data only. Its own go.mod keeps it out of the root module zip that go install downloads; the files stay in the repository and tests read them by path. Never go:embed from here. automation/release/modulesize gates the root module zip size.
module github.com/tmih06/ompig/chord/delta/testdata

go 1.26.0
