package relationships

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	graphMu.RLock()
	ret["graph"] = util.MemoryResult{Memory: util.MemoryUsage(graph), Count: len(graph)}
	graphMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Relationships`, GetMemoryUsage)
}
