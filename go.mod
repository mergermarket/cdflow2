module github.com/mergermarket/cdflow2

go 1.13

require (
	github.com/fsouza/go-dockerclient v1.6.0
	github.com/mergermarket/cdflow2/config v0.0.0-00010101000000-000000000000
	github.com/mergermarket/cdflow2/containers v0.0.0-00010101000000-000000000000
	github.com/mergermarket/cdflow2/release v0.0.0-00010101000000-000000000000
	github.com/mergermarket/cdflow2/terraform v0.0.0-00010101000000-000000000000
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	gopkg.in/yaml.v2 v2.2.5
)

replace github.com/mergermarket/cdflow2/config => ./config

replace github.com/mergermarket/cdflow2/containers => ./containers

replace github.com/mergermarket/cdflow2/release => ./release

replace github.com/mergermarket/cdflow2/terraform => ./terraform
