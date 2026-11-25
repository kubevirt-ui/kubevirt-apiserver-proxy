package namedfilters

import "testing"

func TestMatchVMNADs(t *testing.T) {
	// Bare networkName is namespace-prefixed with the VM's namespace.
	bareName := `{"metadata":{"namespace":"ns1"},"spec":{"template":{"spec":{"networks":[{"multus":{"networkName":"my-nad"}}]}}}}`
	// Already-qualified networkName is used as-is.
	qualified := `{"metadata":{"namespace":"ns1"},"spec":{"template":{"spec":{"networks":[{"multus":{"networkName":"other-ns/my-nad"}}]}}}}`
	// Pod network (no multus) is ignored.
	podNetwork := `{"metadata":{"namespace":"ns1"},"spec":{"template":{"spec":{"networks":[{"pod":{}}]}}}}`
	multiple := `{"metadata":{"namespace":"ns1"},"spec":{"template":{"spec":{"networks":[{"pod":{}},{"multus":{"networkName":"net-a"}},{"multus":{"networkName":"shared/net-b"}}]}}}}`

	tests := []struct {
		name     string
		vm       string
		search   []string
		expected bool
	}{
		{name: "bare name matches namespace-prefixed selection", vm: bareName, search: []string{"ns1/my-nad"}, expected: true},
		{name: "bare name does not match unprefixed selection", vm: bareName, search: []string{"my-nad"}, expected: false},
		{name: "qualified name used as-is", vm: qualified, search: []string{"other-ns/my-nad"}, expected: true},
		{name: "pod network is ignored", vm: podNetwork, search: []string{"ns1/my-nad"}, expected: false},
		{name: "matches any across multiple networks", vm: multiple, search: []string{"nope", "shared/net-b"}, expected: true},
		{name: "no match", vm: multiple, search: []string{"ns1/does-not-exist"}, expected: false},
		{name: "no networks", vm: `{"metadata":{"namespace":"ns1"}}`, search: []string{"ns1/my-nad"}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchVMNADs(vm(tt.vm), tt.search); got != tt.expected {
				t.Errorf("matchVMNADs(%s, %v) = %v, want %v", tt.vm, tt.search, got, tt.expected)
			}
		})
	}
}
