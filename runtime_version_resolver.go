package dotnetcoreruntime

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
)

type RuntimeVersionResolver struct {
	logger LogEmitter
}

func NewRuntimeVersionResolver(logger LogEmitter) RuntimeVersionResolver {
	return RuntimeVersionResolver{logger: logger}
}

func (r RuntimeVersionResolver) Resolve(path string, entry packit.BuildpackPlanEntry, stack string) (postal.Dependency, error) {
	dotnetRuntimeDependencies, defaultVersion, err := filterBuildpackTOML(path, entry.Name, stack)
	if err != nil {
		return postal.Dependency{}, err
	}

	var version string
	if versionStruct, ok := entry.Metadata["version"]; ok {
		version = versionStruct.(string)
	}

	if version == "" || version == "default" {
		version = defaultVersion
	}

	if version == "" {
		version = "*"
	}

	var versionSource string
	if versionSourceStruct, ok := entry.Metadata["version-source"]; ok {
		versionSource = versionSourceStruct.(string)
	}

	constraints, err := gatherVersionConstraints(version, versionSource)
	if err != nil {
		return postal.Dependency{}, err
	}

	var compatibleDependencies []postal.Dependency
	for i, constraint := range constraints {
		if i == 1 { // if 0th constraint not satisfied, no exact match avail
			r.logger.Subprocess("No exact version match found; attempting version roll-forward")
			r.logger.Break()
		}
		for _, dependency := range dotnetRuntimeDependencies {
			depVersion, err := semver.NewVersion(dependency.Version)
			if err != nil {
				return postal.Dependency{}, err
			}

			// create a constraint that the depVersion must be >= requested version to prevent against rolling backwards
			preventRollback, err := semver.NewConstraint(fmt.Sprintf(">= %s", version))
			if err != nil {
				return postal.Dependency{}, err
			}

			if constraint.Check(depVersion) && preventRollback.Check(depVersion) {
				compatibleDependencies = append(compatibleDependencies, dependency)
			}
		}

		// if this constraint can be satisfied, look no further
		if len(compatibleDependencies) > 0 {
			break
		}
	}

	if len(compatibleDependencies) == 0 {
		var supportedVersions []string
		for _, dependency := range dotnetRuntimeDependencies {
			supportedVersions = append(supportedVersions, dependency.Version)
		}

		return postal.Dependency{}, fmt.Errorf(
			"failed to satisfy %q dependency for stack %q with version constraint %q: no compatible versions. Supported versions are: [%s]",
			entry.Name,
			stack,
			version,
			strings.Join(supportedVersions, ", "),
		)
	}

	// makes sure latest version is first in slice
	sort.Slice(compatibleDependencies, func(i, j int) bool {
		iVersion := semver.MustParse(compatibleDependencies[i].Version)
		jVersion := semver.MustParse(compatibleDependencies[j].Version)
		return iVersion.GreaterThan(jVersion)
	})

	return compatibleDependencies[0], nil
}

func containsStack(stacks []string, stack string) bool {
	for _, s := range stacks {
		if s == stack {
			return true
		}
	}
	return false
}

func gatherVersionConstraints(version string, versionSource string) ([]semver.Constraints, error) {
	var constraints []semver.Constraints
	runtimeConstraint, err := semver.NewConstraint(version)
	if err != nil {
		return nil, err
	}
	constraints = append(constraints, *runtimeConstraint)

	// Don't add roll forward constraints if the version source is BP_DOTNET_FRAMEWORK_VERSION or buildpack.yml
	if versionSource != "BP_DOTNET_FRAMEWORK_VERSION" && versionSource != "buildpack.yml" {
		// If version is 1.2.3 or 1.2.* but not 1.2 or 1.*
		if match, _ := regexp.MatchString(`\d+\.\d+\.(\d+$|\*$)`, version); match {
			runtimeVersion, err := semver.NewVersion(strings.TrimSuffix(version, `.*`))
			if err != nil {
				return []semver.Constraints{}, err
			}

			minorConstraint, err := semver.NewConstraint(fmt.Sprintf("%d.%d.*", runtimeVersion.Major(), runtimeVersion.Minor()))
			if err != nil {
				return []semver.Constraints{}, err
			}
			constraints = append(constraints, *minorConstraint)

			majorConstraint, err := semver.NewConstraint(fmt.Sprintf("%d.*", runtimeVersion.Major()))
			if err != nil {
				return []semver.Constraints{}, err
			}
			constraints = append(constraints, *majorConstraint)
		}
	}
	return constraints, nil
}

func filterBuildpackTOML(path, dependencyID, stack string) ([]postal.Dependency, string, error) {
	var buildpackTOML struct {
		Metadata struct {
			DefaultVersions map[string]string   `toml:"default-versions"`
			Dependencies    []postal.Dependency `toml:"dependencies"`
		} `toml:"metadata"`
	}

	_, err := toml.DecodeFile(path, &buildpackTOML)
	if err != nil {
		return []postal.Dependency{}, "", err
	}

	var filteredDependencies []postal.Dependency
	for _, dependency := range buildpackTOML.Metadata.Dependencies {
		if dependency.ID == dependencyID && containsStack(dependency.Stacks, stack) {
			filteredDependencies = append(filteredDependencies, dependency)
		}
	}
	return filteredDependencies, buildpackTOML.Metadata.DefaultVersions[dependencyID], nil
}
