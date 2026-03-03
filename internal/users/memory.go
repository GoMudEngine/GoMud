package users

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {

	ret := map[string]util.MemoryResult{}

	ret["Users"] = util.MemoryResult{Memory: util.MemoryUsage(userManager.Users), Count: len(userManager.Users)}
	ret["Usernames"] = util.MemoryResult{Memory: util.MemoryUsage(userManager.Usernames), Count: len(userManager.Usernames)}
	ret["Connections"] = util.MemoryResult{Memory: util.MemoryUsage(userManager.Connections), Count: len(userManager.Connections)}
	ret["UserConnections"] = util.MemoryResult{Memory: util.MemoryUsage(userManager.UserConnections), Count: len(userManager.UserConnections)}

	return ret
}

func init() {
	util.AddMemoryReporter(`Users`, GetMemoryUsage)
}
