package namedfilters

import (
	"testing"

	"github.com/tidwall/gjson"
)

// vm parses a raw JSON string into a gjson.Result for use as a list item in tests.
func vm(raw string) gjson.Result {
	return gjson.Parse(raw)
}

func TestGetMatcherFor(t *testing.T) {
	// Keys registered for virtualmachines resolve to a matcher.
	for _, key := range []string{osFilterKey, nadFilterKey, schedulingFilterKey, hwDevicesFilterKey, guestAgentFilterKey} {
		if GetMatcherFor(VM, key) == nil {
			t.Errorf("GetMatcherFor(%q, %q) = nil, want a registered handler", VM, key)
		}
	}

	// Yaml paths are not named filters.
	if GetMatcherFor(VM, "metadata.name") != nil {
		t.Errorf("GetMatcherFor(%q, %q) = matcher, want yaml paths to be handled generically", VM, "metadata.name")
	}

	// The VMI OS key resolves to the VMI matcher.
	if GetMatcherFor(VMI, vmiOSFilterKey) == nil {
		t.Errorf("GetMatcherFor(%q, %q) = nil, want a registered handler", VMI, vmiOSFilterKey)
	}

	// Keys are resource-scoped: the VM "os" key is not registered for VMI (and vice versa).
	if GetMatcherFor(VMI, osFilterKey) != nil {
		t.Errorf("GetMatcherFor(%q, %q) = matcher, want VM's key to be unregistered for VMI", VMI, osFilterKey)
	}
	if GetMatcherFor(VM, vmiOSFilterKey) != nil {
		t.Errorf("GetMatcherFor(%q, %q) = matcher, want VMI's key to be unregistered for VM", VM, vmiOSFilterKey)
	}

	// An unregistered resource resolves nothing.
	if GetMatcherFor("templates", osFilterKey) != nil {
		t.Errorf("GetMatcherFor(%q, %q) = matcher, want unregistered resource to resolve nothing", "templates", osFilterKey)
	}
}

func TestIsNonEmptyCollection(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		expected bool
	}{
		{name: "non-empty object", json: `{"a":{"x":1}}`, path: "a", expected: true},
		{name: "empty object", json: `{"a":{}}`, path: "a", expected: false},
		{name: "non-empty array", json: `{"a":[1]}`, path: "a", expected: true},
		{name: "empty array", json: `{"a":[]}`, path: "a", expected: false},
		{name: "missing", json: `{}`, path: "a", expected: false},
		{name: "scalar", json: `{"a":"x"}`, path: "a", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNonEmptyCollection(vm(tt.json).Get(tt.path)); got != tt.expected {
				t.Errorf("isNonEmptyCollection(%s @ %s) = %v, want %v", tt.json, tt.path, got, tt.expected)
			}
		})
	}
}
