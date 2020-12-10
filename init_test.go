package dotnetcoreruntime_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDotnetCoreRuntime(t *testing.T) {
	suite := spec.New("dotnet-core-runtime", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Build", testBuild)
	suite("BuildpackYMLParser", testBuildpackYMLParser)
	suite("Detect", testDetect)
	suite("LogEmitter", testLogEmitter)
	suite("PlanEntryResolver", testPlanEntryResolver)
	suite("PlanRefinery", testPlanRefinery)
	suite("RuntimeVersionResolver", testRuntimeVersionResolver)
	suite("Symlinker", testSymlinker)
	suite.Run(t)
}
