module github.com/paketo-buildpacks/dotnet-core-runtime

go 1.18

replace github.com/paketo-buildpacks/packit/v2 => /Users/caseyj/git/paketo-buildpacks/packit

require (
	github.com/BurntSushi/toml v1.1.0
	github.com/Masterminds/semver v1.5.0
	github.com/onsi/gomega v1.19.0
	github.com/paketo-buildpacks/occam v0.8.0
	github.com/paketo-buildpacks/packit/v2 v2.2.0
	github.com/sclevine/spec v1.4.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/ForestEckhardt/freezer v0.0.10 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/ulikunitz/xz v0.5.10 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/text v0.3.7 // indirect
)
