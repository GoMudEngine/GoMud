package goals

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	cacheMu.RLock()
	ret["cache"] = util.MemoryResult{Memory: util.MemoryUsage(cache), Count: len(cache)}
	ret["nameByMobId"] = util.MemoryResult{Memory: util.MemoryUsage(nameByMobId), Count: len(nameByMobId)}
	cacheMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Goals`, GetMemoryUsage)
}
