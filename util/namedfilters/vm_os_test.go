package namedfilters

import "testing"

func TestMatchVMOS(t *testing.T) {
	tests := []struct {
		name     string
		vm       string
		search   []string
		expected bool
	}{
		{
			name:     "preference name resolves to RHEL",
			vm:       `{"spec":{"preference":{"name":"rhel.9"}}}`,
			search:   []string{"RHEL"},
			expected: true,
		},
		{
			name:     "template annotation resolves to Fedora",
			vm:       `{"spec":{"template":{"metadata":{"annotations":{"vm.kubevirt.io/os":"fedora40"}}}}}`,
			search:   []string{"Fedora"},
			expected: true,
		},
		{
			name:     "os.template label suffix resolves to Windows via win prefix",
			vm:       `{"metadata":{"labels":{"os.template.kubevirt.io/win10":"true"}}}`,
			search:   []string{"Windows"},
			expected: true,
		},
		{
			name:     "windows annotation (win2k22) resolves to Windows",
			vm:       `{"spec":{"template":{"metadata":{"annotations":{"vm.kubevirt.io/os":"win2k22"}}}}}`,
			search:   []string{"Windows"},
			expected: true,
		},
		{
			name:     "centos preference resolves to CentOS",
			vm:       `{"spec":{"preference":{"name":"centos.stream9"}}}`,
			search:   []string{"CentOS"},
			expected: true,
		},
		{
			name:     "unknown OS is treated as Other",
			vm:       `{"metadata":{"name":"mystery"}}`,
			search:   []string{"Other"},
			expected: true,
		},
		{
			name:     "no match when selecting a different OS",
			vm:       `{"spec":{"preference":{"name":"rhel.9"}}}`,
			search:   []string{"Fedora"},
			expected: false,
		},
		{
			name:     "OR across multiple search values",
			vm:       `{"spec":{"preference":{"name":"fedora"}}}`,
			search:   []string{"RHEL", "Fedora"},
			expected: true,
		},
		{
			name:     "case-insensitive match",
			vm:       `{"spec":{"preference":{"name":"rhel.9"}}}`,
			search:   []string{"rhel"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchVMOS(vm(tt.vm), tt.search); got != tt.expected {
				t.Errorf("matchVMOS(%s, %v) = %v, want %v", tt.vm, tt.search, got, tt.expected)
			}
		})
	}
}
