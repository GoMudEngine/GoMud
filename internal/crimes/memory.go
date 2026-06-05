package crimes

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	crimeCacheMu.RLock()
	ret["crimeCache"] = util.MemoryResult{Memory: util.MemoryUsage(crimeCache), Count: len(crimeCache)}
	crimeCacheMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Crimes`, GetMemoryUsage)
}
