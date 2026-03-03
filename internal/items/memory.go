package items

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}

	ret["items"] = util.MemoryResult{Memory: util.MemoryUsage(items), Count: len(items)}

	return ret
}

func init() {
	util.AddMemoryReporter(`Items`, GetMemoryUsage)
}
