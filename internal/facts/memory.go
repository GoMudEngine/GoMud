package facts

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}

	registryMu.RLock()
	count := 0
	if registry != nil {
		count = len(registry.Facts)
	}
	ret["registry"] = util.MemoryResult{Memory: util.MemoryUsage(registry), Count: count}
	registryMu.RUnlock()

	awarenessCacheMu.RLock()
	ret["awarenessCache"] = util.MemoryResult{Memory: util.MemoryUsage(awarenessCache), Count: len(awarenessCache)}
	awarenessCacheMu.RUnlock()

	return ret
}

func init() {
	util.AddMemoryReporter(`Facts`, GetMemoryUsage)
}
