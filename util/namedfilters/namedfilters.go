// Package namedfilters implements the "named" query-key filters the proxy understands (e.g. "os").
//
// Named keys let the UI send a short, stable key and have the proxy own the logic for resolving the
// relevant fields of the resource, instead of encoding yaml paths in the UI.
// Each matcher mirrors the client-side filter predicate so that proxy-filtered and client-filtered results agree.
//
// A matcher is specific to one resource kind because it reads that kind's field layout (a VM's
// `spec.template.spec.*` differs from a VMI's `spec.*`). Matchers are therefore grouped by resource:
// each lives in a file prefixed with its resource (vm_*.go, vmi_*.go) and is wired into the registry
// under that resource's plural.
package namedfilters

import "github.com/tidwall/gjson"

// NamedFilter matches a single list item against the comma-separated searchValues of a named query key
type NamedFilter func(item gjson.Result, searchValues []string) bool

// Resource names as they appear (plural) in the request path
const (
	VM  = "virtualmachines"
	VMI = "virtualmachineinstances"
)

// registry maps a resource plural to its named filters
// named filters are keyed by query key, which must match the UI filter id
// A (resource, key) pair present here is handled by its matcher; anything else is treated as a yaml path by the caller.
var registry = map[string]map[string]NamedFilter{
	VM: {
		osFilterKey:         matchVMOS,
		nadFilterKey:        matchVMNADs,
		schedulingFilterKey: matchVMScheduling,
		hwDevicesFilterKey:  matchVMHWDevices,
		guestAgentFilterKey: matchVMGuestAgent,
	},
	VMI: {
		vmiOSFilterKey: matchVMIOS,
	},
}

// GetMatcherFor returns the matcher registered for the given resource and query key,
// or nil if the (resource, key) pair has no named filter.
func GetMatcherFor(resource, filterKey string) NamedFilter {
	return registry[resource][filterKey]
}

// isNonEmptyCollection reports whether the gjson result is a present object or array with at least
// one entry. Mirrors the UI's `!isEmpty(...)` existence checks for nested objects/arrays.
func isNonEmptyCollection(r gjson.Result) bool {
	if !r.Exists() {
		return false
	}
	if r.IsArray() {
		return len(r.Array()) > 0
	}
	if r.IsObject() {
		return len(r.Map()) > 0
	}
	return false
}
