package namedfilters

import (
	"strings"

	"github.com/tidwall/gjson"
)

const guestAgentFilterKey = "guestAgent"

// Guest agent reporting states, mirroring the UI GuestAgentStatus enum values.
const (
	guestAgentReporting    = "reporting"
	guestAgentNotReporting = "notReporting"
)

const (
	// vmRunningStatus is the status.printableStatus value for a running VM (VM_STATUS.Running).
	vmRunningStatus = "Running"
	// agentConnectedCondition is the status condition type set when the guest agent is connected
	// (VirtualMachineStatusConditionTypes.AgentConnected).
	agentConnectedCondition = "AgentConnected"
)

// matchVMGuestAgent reports whether the VM matches any of the requested guest-agent states.
// Mirrors the UI getGuestAgentFilter: a VM must be Running (status.printableStatus == "Running");
// then "reporting" requires the AgentConnected condition to be True and "notReporting" requires it
// not to be. A non-running VM matches neither. The VM's own status.conditions are read (not the
// VMI's), matching the UI isVMGuestAgentConnected.
func matchVMGuestAgent(item gjson.Result, searchValues []string) bool {
	if item.Get("status.printableStatus").String() != vmRunningStatus {
		return false
	}

	hasAgent := isStatusConditionTrue(item, agentConnectedCondition)

	for _, search := range searchValues {
		switch strings.TrimSpace(search) {
		case guestAgentReporting:
			if hasAgent {
				return true
			}
		case guestAgentNotReporting:
			if !hasAgent {
				return true
			}
		}
	}
	return false
}

// isStatusConditionTrue reports whether item has a status.conditions entry of the given type whose
// status is "True". Mirrors the UI isStatusConditionTrue (first condition of that type wins).
func isStatusConditionTrue(item gjson.Result, condType string) bool {
	for _, cond := range item.Get("status.conditions").Array() {
		if cond.Get("type").String() == condType {
			return cond.Get("status").String() == "True"
		}
	}
	return false
}
