package knowledge

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	knowledgeCacheMu.RLock()
	ret["knowledgeCache"] = util.MemoryResult{Memory: util.MemoryUsage(knowledgeCache), Count: len(knowledgeCache)}
	knowledgeCacheMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Knowledge`, GetMemoryUsage)
}
