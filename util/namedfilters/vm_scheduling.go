package namedfilters

import (
	"strings"

	"github.com/tidwall/gjson"
)

const schedulingFilterKey = "scheduling"

// Scheduling kinds, mirroring the UI SchedulingKind enum values.
const (
	schedulingAffinityRules = "affinityRules"
	schedulingNodeSelector  = "nodeSelector"
)

// Mirrors getSchedulingFilter.match on the UI
func matchVMScheduling(item gjson.Result, searchValues []string) bool {
	for _, search := range searchValues {
		switch strings.TrimSpace(search) {
		case schedulingAffinityRules:
			if isNonEmptyCollection(item.Get("spec.template.spec.affinity")) {
				return true
			}
		case schedulingNodeSelector:
			if isNonEmptyCollection(item.Get("spec.template.spec.nodeSelector")) {
				return true
			}
		}
	}
	return false
}
