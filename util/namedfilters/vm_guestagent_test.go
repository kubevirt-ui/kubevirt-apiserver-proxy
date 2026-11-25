package namedfilters

import "testing"

func TestMatchVMGuestAgent(t *testing.T) {
	runningWithAgent := `{"status":{"printableStatus":"Running","conditions":[{"type":"AgentConnected","status":"True"}]}}`
	runningAgentFalse := `{"status":{"printableStatus":"Running","conditions":[{"type":"AgentConnected","status":"False"}]}}`
	runningNoAgentCond := `{"status":{"printableStatus":"Running","conditions":[{"type":"Ready","status":"True"}]}}`
	stoppedWithAgent := `{"status":{"printableStatus":"Stopped","conditions":[{"type":"AgentConnected","status":"True"}]}}`
	noStatus := `{"metadata":{"name":"vm"}}`

	tests := []struct {
		name     string
		vm       string
		search   []string
		expected bool
	}{
		{name: "running + agent connected, reporting selected", vm: runningWithAgent, search: []string{guestAgentReporting}, expected: true},
		{name: "running + agent connected, notReporting selected", vm: runningWithAgent, search: []string{guestAgentNotReporting}, expected: false},
		{name: "running + agent condition False, notReporting selected", vm: runningAgentFalse, search: []string{guestAgentNotReporting}, expected: true},
		{name: "running + agent condition False, reporting selected", vm: runningAgentFalse, search: []string{guestAgentReporting}, expected: false},
		{name: "running + no agent condition, notReporting selected", vm: runningNoAgentCond, search: []string{guestAgentNotReporting}, expected: true},
		{name: "running + no agent condition, reporting selected", vm: runningNoAgentCond, search: []string{guestAgentReporting}, expected: false},
		{name: "not running never matches reporting", vm: stoppedWithAgent, search: []string{guestAgentReporting}, expected: false},
		{name: "not running never matches notReporting", vm: stoppedWithAgent, search: []string{guestAgentNotReporting}, expected: false},
		{name: "no status is not running", vm: noStatus, search: []string{guestAgentNotReporting}, expected: false},
		{name: "OR across both states (running + agent)", vm: runningWithAgent, search: []string{guestAgentReporting, guestAgentNotReporting}, expected: true},
		{name: "unknown search value does not match", vm: runningWithAgent, search: []string{"bogus"}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchVMGuestAgent(vm(tt.vm), tt.search); got != tt.expected {
				t.Errorf("matchVMGuestAgent(%s, %v) = %v, want %v", tt.vm, tt.search, got, tt.expected)
			}
		})
	}
}
