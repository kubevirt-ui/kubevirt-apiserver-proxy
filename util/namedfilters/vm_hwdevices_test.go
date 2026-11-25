package namedfilters

import "testing"

func TestMatchVMHWDevices(t *testing.T) {
	gpus := `{"spec":{"template":{"spec":{"domain":{"devices":{"gpus":[{"name":"gpu1"}]}}}}}}`
	hostDevices := `{"spec":{"template":{"spec":{"domain":{"devices":{"hostDevices":[{"name":"hd1"}]}}}}}}`
	neither := `{"spec":{"template":{"spec":{"domain":{"devices":{}}}}}}`
	emptyGPUs := `{"spec":{"template":{"spec":{"domain":{"devices":{"gpus":[]}}}}}}`

	tests := []struct {
		name     string
		vm       string
		search   []string
		expected bool
	}{
		{name: "gpus set, gpu selected", vm: gpus, search: []string{hwDeviceGPU}, expected: true},
		{name: "hostDevices set, host selected", vm: hostDevices, search: []string{hwDeviceHost}, expected: true},
		{name: "gpus set, host selected", vm: gpus, search: []string{hwDeviceHost}, expected: false},
		{name: "OR across kinds", vm: hostDevices, search: []string{hwDeviceGPU, hwDeviceHost}, expected: true},
		{name: "neither set", vm: neither, search: []string{hwDeviceGPU, hwDeviceHost}, expected: false},
		{name: "empty gpus array does not match", vm: emptyGPUs, search: []string{hwDeviceGPU}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchVMHWDevices(vm(tt.vm), tt.search); got != tt.expected {
				t.Errorf("matchVMHWDevices(%s, %v) = %v, want %v", tt.vm, tt.search, got, tt.expected)
			}
		})
	}
}
