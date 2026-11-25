package namedfilters

import "testing"

func TestMatchVMIOS(t *testing.T) {
	tests := []struct {
		name     string
		vmi      string
		search   []string
		expected bool
	}{
		{
			name:     "os annotation resolves to Fedora",
			vmi:      `{"metadata":{"annotations":{"vm.kubevirt.io/os":"fedora40"}}}`,
			search:   []string{"Fedora"},
			expected: true,
		},
		{
			name:     "windows annotation (win2k22) resolves to Windows via win prefix",
			vmi:      `{"metadata":{"annotations":{"vm.kubevirt.io/os":"win2k22"}}}`,
			search:   []string{"Windows"},
			expected: true,
		},
		{
			name:     "rhel annotation resolves to RHEL",
			vmi:      `{"metadata":{"annotations":{"vm.kubevirt.io/os":"rhel9.4"}}}`,
			search:   []string{"RHEL"},
			expected: true,
		},
		{
			name:     "no annotation is treated as Other",
			vmi:      `{"metadata":{"name":"mystery"}}`,
			search:   []string{"Other"},
			expected: true,
		},
		{
			name:     "unknown os is treated as Other",
			vmi:      `{"metadata":{"annotations":{"vm.kubevirt.io/os":"haiku"}}}`,
			search:   []string{"Other"},
			expected: true,
		},
		{
			name:     "no match when selecting a different OS",
			vmi:      `{"metadata":{"annotations":{"vm.kubevirt.io/os":"fedora40"}}}`,
			search:   []string{"RHEL"},
			expected: false,
		},
		{
			name:     "OR across multiple search values",
			vmi:      `{"metadata":{"annotations":{"vm.kubevirt.io/os":"fedora40"}}}`,
			search:   []string{"RHEL", "Fedora"},
			expected: true,
		},
		{
			name:     "case-insensitive match",
			vmi:      `{"metadata":{"annotations":{"vm.kubevirt.io/os":"fedora40"}}}`,
			search:   []string{"fedora"},
			expected: true,
		},
		{
			name:     "VMI OS does not read VM spec.template paths",
			vmi:      `{"spec":{"template":{"metadata":{"annotations":{"vm.kubevirt.io/os":"fedora40"}}}}}`,
			search:   []string{"Fedora"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchVMIOS(vm(tt.vmi), tt.search); got != tt.expected {
				t.Errorf("matchVMIOS(%s, %v) = %v, want %v", tt.vmi, tt.search, got, tt.expected)
			}
		})
	}
}
