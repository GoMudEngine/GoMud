package factions

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	definitionsMu.RLock()
	ret["definitions"] = util.MemoryResult{Memory: util.MemoryUsage(definitions), Count: len(definitions)}
	definitionsMu.RUnlock()
	repCacheMu.RLock()
	ret["repCache"] = util.MemoryResult{Memory: util.MemoryUsage(repCache), Count: len(repCache)}
	repCacheMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Factions`, GetMemoryUsage)
}
