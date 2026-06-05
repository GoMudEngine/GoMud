package opinions

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	opinionCacheMu.RLock()
	ret["opinionCache"] = util.MemoryResult{Memory: util.MemoryUsage(opinionCache), Count: len(opinionCache)}
	ret["nameByMobId"] = util.MemoryResult{Memory: util.MemoryUsage(nameByMobId), Count: len(nameByMobId)}
	opinionCacheMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Opinions`, GetMemoryUsage)
}
