package namedfilters

import (
	"github.com/tidwall/gjson"
)

const vmiOSFilterKey = "vmi-os"

// matchVMIOS reports whether the VMI's resolved OS label is one of the requested values.
// Mirrors the UI VMI OS filter: selected.includes(getOSLabel(vmi) ?? "Other"), where
// getOSLabel(vmi) = getMatchingOSLabel(getAnnotation(vmi, "vm.kubevirt.io/os")).
//
// Unlike a VM, a VMI carries the OS annotation directly on its own metadata (no spec.template and no
// preference), so resolution is a single annotation lookup rather than getVMOSLabel's cascade.
func matchVMIOS(item gjson.Result, searchValues []string) bool {
	osAnnotation := item.Get(`metadata.annotations.` + vmOSAnnotation).String()
	return osLabelMatches(getMatchingOSLabel(osAnnotation), searchValues)
}
