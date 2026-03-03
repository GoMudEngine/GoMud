package mobs

import (
	"github.com/GoMudEngine/GoMud/internal/util"
)

func GetMemoryUsage() map[string]util.MemoryResult {

	ret := map[string]util.MemoryResult{}

	ret["mobs"] = util.MemoryResult{Memory: util.MemoryUsage(mobs), Count: len(mobs)}
	ret["allMobNames"] = util.MemoryResult{Memory: util.MemoryUsage(allMobNames), Count: len(allMobNames)}
	ret["mobInstances"] = util.MemoryResult{Memory: util.MemoryUsage(mobInstances), Count: len(mobInstances)}
	ret["mobsHatePlayers"] = util.MemoryResult{Memory: util.MemoryUsage(mobsHatePlayers), Count: len(mobsHatePlayers)}

	return ret
}

func init() {
	util.AddMemoryReporter(`Mobs`, GetMemoryUsage)
}
