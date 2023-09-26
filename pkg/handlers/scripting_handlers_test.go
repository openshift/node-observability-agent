package handlers

import (
	"testing"

	"github.com/openshift/node-observability-agent/pkg/connectors"
	"github.com/openshift/node-observability-agent/pkg/runs"
)

// TestScriptingHandlers
func TestScriptingHandlers(t *testing.T) {
	// cover the pass scenario
	t.Run("ScriptingHandlers : should pass", func(t *testing.T) {
		h := NewScriptingHandlers("/tmp", "TEST")
		connector := connectors.Connector{}
		connector.Prepare("sh", []string{"-c", "ls -la"})
		run := h.executeScript("12345678", &connector)
		if !run.Successful || run.Type != runs.ScriptingRun {
			t.Errorf("Expecting execution of script to pass, but got %q", run.Error)
		}
	})

	// cover the fail scenario
	t.Run("ScriptingHandlers : should fail", func(t *testing.T) {
		h := NewScriptingHandlers("/tmp", "TEST")
		connector := connectors.Connector{}
		connector.Prepare("sh", []string{"-c", "no-script-to-execute"})
		run := h.executeScript("12345678", &connector)
		if run.Successful || run.Type != runs.ScriptingRun {
			t.Errorf("Expecting execution of script to fail, but got %v", run.Successful)
		}
	})
}
