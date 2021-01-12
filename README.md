# Dotnet Core Runtime Cloud Native Buildpack

The Dotnet Core Runtime CNB provides a version of the [Dotnet Core
Runtime](https://github.com/dotnet/runtime) and sets the initial `$DOTNET_ROOT`
location.

A usage example can be found in the
[`samples` repository under the `dotnet-core/runtime`
directory](https://github.com/paketo-buildpacks/samples/tree/main/dotnet-core/runtime).

## Integration

The Dotnet Core Runtime CNB provides dotnet-runtime as a dependency.
Downstream buildpacks, like [Dotnet
Publish](https://github.com/paketo-buildpacks/dotnet-publish) and [Dotnet
Execute](https://github.com/paketo-buildpacks/dotnet-execute) can require the
dotnet-runtime dependency by generating a [Build Plan
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
    # the runtimes instead use the dotnet-core-sdk. However,
    # if you are writing a buildpack that needs to use the dotnet core runtime during
    # its build process, this flag should be set to true.
    build = true

    # Setting the launch flag to true will ensure that the Dotnet Core Runtime
    # dependency is available on the $DOTNET_ROOT for the running application. If you are
    # writing an application that needs to run Dotnet Core Runtime at runtime, this flag should
    # be set to true.
    launch = true
```

### Specifying runtime versions

#### Self contained applications & Framework dependent applications
Be aware that specifying a dotnet runtime version for both framework dependent
deployments and self contained deployments may result in errors if the
selected runtimes do not match those used to build the application.

#### Source based applications
We do not recommend specifying a runtime version for source based workflows.
Doing so could result in an incompatibility between the dotnet-sdk and
its internal dotnet-runtime.

## Usage

To package this buildpack for consumption

```
$ ./scripts/package.sh
```

This builds the buildpack's Go source using `GOOS=linux` by default. You can
supply another value as the first argument to `package.sh`.

## `buildpack.yml` Configurations

```yaml
dotnet-framework:
  # this allows you to specify a version constaint for the dotnet-runtime dependency
  # any valid semver constaints (e.g. 2.* and 2.1.*) are also acceptable. Including
  # **any** dotnet-framework version entry in the buildpack.yml will prevent the
  # buildpack from running version roll-forward logic
  version: "2.1.14"
```
For more information about version roll-forward logic, see [the .NET
documentation.](https://docs.microsoft.com/en-us/dotnet/core/versions/selection#framework-dependent-apps-roll-forward)
