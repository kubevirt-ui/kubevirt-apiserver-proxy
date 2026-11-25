package namedfilters

import (
	"strings"

	"github.com/tidwall/gjson"
)

const nadFilterKey = "nad"

// matchVMNADs reports whether the VM is attached to any of the requested NetworkAttachmentDefinitions.
// Mirrors the UI getNADsFilter: collect each network's multus.networkName, namespace-prefix the
// bare ones (`<vm namespace>/<name>`), and match (exact) if any is in the selected values.
func matchVMNADs(item gjson.Result, searchValues []string) bool {
	networks := item.Get("spec.template.spec.networks").Array()
	if len(networks) == 0 {
		return false
	}

	namespace := item.Get("metadata.namespace").String()

	selected := make(map[string]struct{}, len(searchValues))
	for _, s := range searchValues {
		selected[s] = struct{}{}
	}

	for _, network := range networks {
		networkName := network.Get("multus.networkName").String()
		if networkName == "" {
			continue
		}
		if !strings.Contains(networkName, "/") {
			networkName = namespace + "/" + networkName
		}
		if _, ok := selected[networkName]; ok {
			return true
		}
	}
	return false
}
