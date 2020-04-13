# Dotnet Core Runtime Cloud Native Buildpack

## Integration

The Dotnet Core Runtime CNB provides the dotnet core runtime as a dependency. Downstream buildpacks, like
[Dotnet Core Build](https://github.com/paketo-buildpacks/dotnet-core-build) or
by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the Dotnet Core Runtime dependency is "dotnet-runtime". This value is considered
  # part of the public API for the buildpack and will not change without a plan
  # for deprecation.
  name = "dotnet-runtime"

  # The version of the Dotnet Core Runtime dependency is not required. In the case it
  # is not specified, the buildpack will provide the default version, which can
  # be seen in the buildpack.toml file.
  # If you wish to request a specific version, the buildpack supports
  # specifying a semver constraint in the form of "3.*", "3.1.*", or even
  # "3.1.1".
  version = "3.1.1"

  # The Dotnet Core Runtime buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the Dotnet Core Runtime
    # dependency is available to subsequent buildpacks during their build phase.
    # Currently we do not recommend having your application directly interface with
    # the runtimes instead use the dotnet-core-sdk-cnb. However,
    # if you are writing a buildpack that needs to use the dotnet core runtime during
    # its build process, this flag should be set to true.
    build = true
```

### Specifying runtime versions

#### Self contained applications & Framework dependent applications
Be aware that specifying a dotnet runtime version for both framework dependent
deployments and self contained deployments  may result in errors if the
selected runtimes do not match those used to build the application.

#### Source based applications
We do not recommend specifying a runtime version for source based workflows.
Doing so could result in an incompatibility between the dotnet-sdk and
it's internal dotnet-runtime.

## Usage

To package this buildpack for consumption

```
$ ./scripts/package.sh
```

This builds the buildpack's Go source using `GOOS=linux` by default. You can
supply another value as the first argument to `package.sh`.
