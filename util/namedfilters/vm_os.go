package namedfilters

import (
	"strings"

	"github.com/tidwall/gjson"
)

const osFilterKey = "os"

const (
	osWindowsLabel = "Windows"
	// Windows OS terms use the "win" prefix (e.g. "win2k22"), so match on that rather than the label.
	osWindowsPrefix = "win"
	// Fallback label when no known OS matches, mirroring the UI (getOSLabel(vm) ?? OS_NAME_LABELS.other).
	osOtherLabel = "Other"

	vmOSAnnotation           = `vm\.kubevirt\.io/os`
	osTemplateLabelPrefix    = "os.template.kubevirt.io"
	nameOSTemplateAnnotation = "name.os.template.kubevirt.io"
)

// osNameLabels mirrors the UI OS_NAME_LABELS values in the same order (Object.values insertion
// order). getMatchingOSLabel returns the first label in this order whose prefix matches a term.
var osNameLabels = []string{"CentOS", "Fedora", "Other", "RHEL", "Windows"}

// matchVMOS reports whether the VM's resolved OS label is one of the requested values.
// Mirrors the UI getOSFilter: selected.includes(getOSLabel(vm) ?? "Other").
func matchVMOS(item gjson.Result, searchValues []string) bool {
	return osLabelMatches(getVMOSLabel(item), searchValues)
}

// osLabelMatches reports whether osLabel equals any of the requested values (case-insensitive). An
// empty label is treated as "Other", mirroring the UI `getOSLabel(...) ?? OS_NAME_LABELS.other`.
// Shared by the VM and VMI OS matchers.
func osLabelMatches(osLabel string, searchValues []string) bool {
	if osLabel == "" {
		osLabel = osOtherLabel
	}
	for _, search := range searchValues {
		if strings.EqualFold(osLabel, strings.TrimSpace(search)) {
			return true
		}
	}
	return false
}

// getVMOSLabel resolves a VM's canonical OS label (one of osNameLabels) from the OS annotation, OS
// label and preference name. Mirrors the UI getOSLabel.
func getVMOSLabel(item gjson.Result) string {
	osAnnotation := item.Get(`spec.template.metadata.annotations.` + vmOSAnnotation).String()

	osLabel := getVMOperatingSystemName(item)
	if osLabel == "" {
		osLabel = getVMOperatingSystem(item)
	}

	osPreference := item.Get("spec.preference.name").String()

	return getMatchingOSLabel(osAnnotation, osLabel, osPreference)
}

// getVMOperatingSystem mirrors the UI selector: the suffix of the first os.template.kubevirt.io/<os>
// label, falling back to the vm.kubevirt.io/os annotation on the VM metadata.
func getVMOperatingSystem(item gjson.Result) string {
	if suffix := findKeySuffixValue(item.Get("metadata.labels"), osTemplateLabelPrefix); suffix != "" {
		return suffix
	}
	return item.Get(`metadata.annotations.` + vmOSAnnotation).String()
}

// getVMOperatingSystemName mirrors the UI selector: the value of the first
// name.os.template.kubevirt.io/<os> annotation on the VM metadata.
func getVMOperatingSystemName(item gjson.Result) string {
	os := getVMOperatingSystem(item)
	if os == "" {
		return ""
	}
	return valueByKeyPrefix(item.Get("metadata.annotations"), nameOSTemplateAnnotation+"/"+os)
}

// getMatchingOSLabel returns the first osNameLabels entry whose prefix matches any of the given
// terms (case-insensitive startsWith). Windows uses the "win" prefix. Mirrors the UI
// getMatchingOSLabel. Returns "" when no known OS matches.
func getMatchingOSLabel(terms ...string) string {
	for _, label := range osNameLabels {
		prefix := label
		if label == osWindowsLabel {
			prefix = osWindowsPrefix
		}
		prefix = strings.ToLower(prefix)

		for _, term := range terms {
			if term != "" && strings.HasPrefix(strings.ToLower(term), prefix) {
				return label
			}
		}
	}
	return ""
}

// findKeySuffixValue returns the substring after the last '/' of the first key in obj that starts
// with keyPrefix. Mirrors the UI findKeySuffixValue (returns "" when the key has no '/').
func findKeySuffixValue(obj gjson.Result, keyPrefix string) string {
	key := firstKeyWithPrefix(obj, keyPrefix)
	if idx := strings.LastIndex(key, "/"); idx > 0 {
		return key[idx+1:]
	}
	return ""
}

// valueByKeyPrefix returns the value of the first key in obj that starts with keyPrefix.
// Mirrors the UI getValueByPrefix.
func valueByKeyPrefix(obj gjson.Result, keyPrefix string) string {
	if !obj.IsObject() {
		return ""
	}
	result := ""
	obj.ForEach(func(key, value gjson.Result) bool {
		if strings.HasPrefix(key.String(), keyPrefix) {
			result = value.String()
			return false // stop at the first match, matching Array.prototype.find
		}
		return true
	})
	return result
}

// firstKeyWithPrefix returns the first key in obj that starts with keyPrefix, or "".
func firstKeyWithPrefix(obj gjson.Result, keyPrefix string) string {
	if !obj.IsObject() {
		return ""
	}
	result := ""
	obj.ForEach(func(key, _ gjson.Result) bool {
		if strings.HasPrefix(key.String(), keyPrefix) {
			result = key.String()
			return false // stop at the first match, matching Array.prototype.find
		}
		return true
	})
	return result
}
