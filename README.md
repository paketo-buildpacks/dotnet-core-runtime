# .NET Core Runtime Cloud Native Buildpack

The .NET Core Runtime CNB provides a version of the [.NET Core
Runtime](https://github.com/dotnet/runtime) and sets the initial `$DOTNET_ROOT`
location.

A usage example can be found in the
[`samples` repository under the `dotnet-core/runtime`
directory](https://github.com/paketo-buildpacks/samples/tree/main/dotnet-core/runtime).

## Integration

The .NET Core Runtime CNB provides `dotnet-runtime` as a dependency.
Downstream buildpacks, like [Dotnet
Publish](https://github.com/paketo-buildpacks/dotnet-publish) and [Dotnet
Execute](https://github.com/paketo-buildpacks/dotnet-execute) can require the
`dotnet-runtime` dependency by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the .NET Core Runtime dependency is "dotnet-runtime". This value is considered
  # part of the public API for the buildpack and will not change without a plan
  # for deprecation.
  name = "dotnet-runtime"

  # The version of the .NET Core Runtime dependency is not required. In the case it
  # is not specified, the buildpack will provide the default version, which can
  # be seen in the buildpack.toml file.
  # If you wish to request a specific version, the buildpack supports
  # specifying a semver constraint in the form of "3.*", "3.1.*", or even
  # "3.1.1".
  version = "3.1.1"

  # The .NET Core Runtime buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the .NET Core Runtime
    # dependency is available to subsequent buildpacks during their build phase.
    # Currently we do not recommend having your application directly interface with
    # the runtimes instead use the dotnet-core-sdk. However,
    # if you are writing a buildpack that needs to use the dotnet core runtime during
    # its build process, this flag should be set to true.
    build = true

    # Setting the launch flag to true will ensure that the .NET Core Runtime
    # dependency is available on the $DOTNET_ROOT for the running application. If you are
    # writing an application that needs to run .NET Core Runtime at runtime, this flag should
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
Doing so could result in an incompatibility between the `dotnet-sdk` and
its internal `dotnet-runtime`.

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh -v <version>
```

## Configuration

Specifying the .NET Framework Version through `buildpack.yml` configuration
will be deprecated in .NET Core Runtime Buildpack v1.0.0.

To migrate from using `buildpack.yml` please set the following environment
variables at build time either directly (ex. `pack build my-app --env
BP_ENVIRONMENT_VARIABLE=some-value`) or through [a `project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md).

### `BP_DOTNET_FRAMEWORK_VERSION`
The `BP_DOTNET_FRAMEWORK_VERSION` variable allows you to specify the version of .NET Core Runtime that is installed.

```shell
BP_DOTNET_FRAMEWORK_VERSION=5.0.4
```

This will replace the following structure in `buildpack.yml`:
```yaml
dotnet-framework:
  version: "5.0.4"
```
For more information about version roll-forward logic, see [the .NET
documentation.](https://docs.microsoft.com/en-us/dotnet/core/versions/selection#framework-dependent-apps-roll-forward)

### `BP_DOTNET_ROLL_FORWARD`
The `BP_DOTNET_ROLL_FORWARD` variable, when set to `Disable`, will only allow binding to the exact version specified.
See [.NET Core Runtime Binding](https://github.com/dotnet/designs/blob/main/accepted/2019/runtime-binding.md#rollforward) for more information.

This variable has no purpose when using either `buildpack.yml` or `BP_DOTNET_FRAMEWORK_VERSION` to set the version, since those methods allow wildcarded version specifications. 
The version must be set using either the `.runtimeconfig.json` or `vb|fs|csproj` files. 
