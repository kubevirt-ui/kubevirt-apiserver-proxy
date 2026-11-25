package namedfilters

import "testing"

func TestMatchVMScheduling(t *testing.T) {
	affinity := `{"spec":{"template":{"spec":{"affinity":{"nodeAffinity":{}}}}}}`
	nodeSelector := `{"spec":{"template":{"spec":{"nodeSelector":{"disktype":"ssd"}}}}}`
	both := `{"spec":{"template":{"spec":{"affinity":{"podAffinity":{}},"nodeSelector":{"disktype":"ssd"}}}}}`
	neither := `{"spec":{"template":{"spec":{}}}}`
	emptyAffinity := `{"spec":{"template":{"spec":{"affinity":{}}}}}`

	tests := []struct {
		name     string
		vm       string
		search   []string
		expected bool
	}{
		{name: "affinity set, affinity selected", vm: affinity, search: []string{schedulingAffinityRules}, expected: true},
		{name: "nodeSelector set, nodeSelector selected", vm: nodeSelector, search: []string{schedulingNodeSelector}, expected: true},
		{name: "affinity set, nodeSelector selected", vm: affinity, search: []string{schedulingNodeSelector}, expected: false},
		{name: "both set, OR matches", vm: both, search: []string{schedulingAffinityRules, schedulingNodeSelector}, expected: true},
		{name: "neither set", vm: neither, search: []string{schedulingAffinityRules, schedulingNodeSelector}, expected: false},
		{name: "empty affinity object does not match", vm: emptyAffinity, search: []string{schedulingAffinityRules}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchVMScheduling(vm(tt.vm), tt.search); got != tt.expected {
				t.Errorf("matchVMScheduling(%s, %v) = %v, want %v", tt.vm, tt.search, got, tt.expected)
			}
		})
	}
}
