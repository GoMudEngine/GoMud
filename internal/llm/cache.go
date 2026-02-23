package llm

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	cacheMu   sync.Mutex
	respCache = map[string]cacheEntry{} // key: "mobInstanceId:topic"

	pendingMu sync.Mutex
	pending   = map[int]bool{} // key: mobInstanceId
)

type cacheEntry struct {
	response string
	expiry   uint64 // round number at which entry expires
}

func cacheKey(mobInstanceId int, topic string) string {
	return fmt.Sprintf("%d:%s", mobInstanceId, topic)
}

// checkCache returns a cached response if present and unexpired.
func checkCache(mobInstanceId int, topic string) (string, bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	entry, ok := respCache[cacheKey(mobInstanceId, topic)]
	if !ok {
		return "", false
	}
	if util.GetRoundCount() > entry.expiry {
		delete(respCache, cacheKey(mobInstanceId, topic))
		return "", false
	}
	return entry.response, true
}

// storeCache saves a response for mobInstanceId+topic with the given TTL.
// ttl uses gametime period syntax (e.g. "1h"). Falls back to a large value if empty.
func storeCache(mobInstanceId int, topic, response string, ttl string) {
	if ttl == "" {
		ttl = "1h"
	}
	baseline := gametime.GetDate(1000000)
	expiryRound := baseline.AddPeriod(ttl)
	delta := expiryRound - 1000000

	cacheMu.Lock()
	defer cacheMu.Unlock()
	respCache[cacheKey(mobInstanceId, topic)] = cacheEntry{
		response: response,
		expiry:   util.GetRoundCount() + delta,
	}
}

// isPending returns true if there is already an in-flight request for this mob.
func isPending(mobInstanceId int) bool {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	return pending[mobInstanceId]
}

// setPending marks or clears the in-flight flag for a mob.
func setPending(mobInstanceId int, val bool) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	if val {
		pending[mobInstanceId] = true
	} else {
		delete(pending, mobInstanceId)
	}
}
