package bounties

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	registryMu.RLock()
	count := 0
	if registry != nil {
		count = len(registry.Bounties)
	}
	ret["registry"] = util.MemoryResult{Memory: util.MemoryUsage(registry), Count: count}
	registryMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Bounties`, GetMemoryUsage)
}
