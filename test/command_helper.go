package test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// CommandTest struct to define a test scenario
type CommandTest struct {
	Setup               func()
	Teardown            func()
	Cmd                 func() *cobra.Command
	CmdArgs             []string
	ProcessOutput       func(t *testing.T, s string)
	ShouldContainErrors []string
	HasCustomError      error
	ShouldContain       []string
	ShouldNotContain    []string
}

// RunCommandTests runs all set test scenarios
// *** USAGE ***
//
//	func TestFooCommands(t *testing.T) {
//		scenarios := make(map[string]test.CommandTest)
//		scenarios["'Foo' command should throw error without args"] =
//			test.CommandTest{
//				Cmd:     NewRootCommand,
//				CmdArgs: []string{"foo"},11
//				ShouldContainErrors: []string{`No args set!`},
//			}
//		scenarios["'Foo' command run successfully with args"] =
//		test.CommandTest{
//			Cmd:     NewRootCommand,
//			CmdArgs: []string{"foo","bar"},11
//			ShouldContain: []string{`Run smoothly!`},
//		}
//
// // Both tests will run
//
//		test.RunCommandTests(t, scenarios)
//	}
func RunCommandTests(t *testing.T, cmdTests map[string]CommandTest) {
	for description, cmdTest := range cmdTests {
		t.Run(description, func(t *testing.T) {
			runCommandTest(t, cmdTest)
		})
	}
}

func runCommandTest(t *testing.T, v CommandTest) {
	// Arrange
	if v.Setup != nil {
		v.Setup()
	}

	// Act
	if v.Cmd == nil {
		t.Error("Cmd attribute not found")
	}
	cmd := v.Cmd()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := ExecuteCommand(cmd, v.CmdArgs...)
	if v.ProcessOutput != nil {
		v.ProcessOutput(t, output)
	}

	// Assert
	if len(v.ShouldContainErrors) == 0 && v.HasCustomError != nil {
		assert.NoError(t, err)
	}
	if len(v.ShouldContainErrors) != 0 {
		assert.Error(t, err)
		for _, msg := range v.ShouldContainErrors {
			assert.EqualError(t, err, msg)
		}
	}
	if v.HasCustomError != nil {
		assert.True(t, errors.Is(err, v.HasCustomError))
	}

	for _, expectedString := range v.ShouldContain {
		assert.Contains(t, output, expectedString)
	}
	for _, unexpectedString := range v.ShouldNotContain {
		assert.NotContains(t, output, unexpectedString)
	}

	if v.Teardown != nil {
		v.Teardown()
	}
}
