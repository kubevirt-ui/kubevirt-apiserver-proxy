package namedfilters

import (
	"strings"

	"github.com/tidwall/gjson"
)

const hwDevicesFilterKey = "hwDevices"

// Hardware device kinds, mirroring the UI HWDeviceKind enum values.
const (
	hwDeviceGPU  = "gpu"
	hwDeviceHost = "host"
)

// Mirrors getHWDevicesFilter.match on the UI
func matchVMHWDevices(item gjson.Result, searchValues []string) bool {
	for _, search := range searchValues {
		switch strings.TrimSpace(search) {
		case hwDeviceGPU:
			if isNonEmptyCollection(item.Get("spec.template.spec.domain.devices.gpus")) {
				return true
			}
		case hwDeviceHost:
			if isNonEmptyCollection(item.Get("spec.template.spec.domain.devices.hostDevices")) {
				return true
			}
		}
	}
	return false
}
