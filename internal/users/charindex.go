package users

import (
	"bytes"
	"strings"
	"sync"
)

// CharacterIndex is an in-memory index mapping lowercase character names to
// user IDs. One user ID may own multiple character names (active character plus
// any stored alts). The index is rebuilt at startup from user records; the
// alt-characters module is responsible for populating alt names.
type CharacterIndex struct {
	mu     sync.RWMutex
	byName map[string]int // lowercase character name -> userId
}

var characterIndex = &CharacterIndex{
	byName: make(map[string]int),
}

// GetCharacterIndex returns the singleton CharacterIndex.
func GetCharacterIndex() *CharacterIndex {
	return characterIndex
}

// Add registers a character name as belonging to userId. The name is
// normalized to lowercase before storage. If the name is already present it
// is overwritten with the new userId.
func (ci *CharacterIndex) Add(name string, userId int) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.byName[strings.ToLower(name)] = userId
}

// Remove deletes the entry for name from the index. It is a no-op if the
// name is not present.
func (ci *CharacterIndex) Remove(name string) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	delete(ci.byName, strings.ToLower(name))
}

// Len returns the number of character names currently in the index.
func (ci *CharacterIndex) Len() int {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return len(ci.byName)
}

// ForEach calls fn for every entry in the index in unspecified order.
// Returning false from fn stops iteration. The index is read-locked for the
// duration of the call; fn must not call any CharacterIndex method.
func (ci *CharacterIndex) ForEach(fn func(name string, userId int) bool) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	for name, userId := range ci.byName {
		if !fn(name, userId) {
			return
		}
	}
}

// Find looks up a character name and returns the owning userId. The lookup is
// case-insensitive. Returns (0, false) when the name is not found.
func (ci *CharacterIndex) Find(name string) (userId int, found bool) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	userId, found = ci.byName[strings.ToLower(name)]
	return
}

// Rebuild clears the index and repopulates it from every user file on disk
// plus all currently online users. Only active character names are added here;
// the alt-characters module is responsible for adding alt names after this
// runs.
func (ci *CharacterIndex) Rebuild() {
	ci.RebuildFromScan(ScanUserFiles())
}

// RebuildFromIndex repopulates the character index straight from the user
// index records plus all currently online users, without opening a single
// user file.
func (ci *CharacterIndex) RebuildFromIndex(idx *UserIndex) {
	newMap := make(map[string]int)

	idx.ForEachRecord(func(rec IndexUserRecord) bool {
		if name := string(bytes.TrimRight(rec.CharacterName[:], "\x00")); name != `` {
			newMap[strings.ToLower(name)] = int(rec.UserID)
		}
		return true
	})

	for _, u := range GetAllActiveUsers() {
		if u.Character != nil && u.Character.Name != "" {
			newMap[strings.ToLower(u.Character.Name)] = u.UserId
		}
	}

	ci.mu.Lock()
	ci.byName = newMap
	ci.mu.Unlock()
}

// RebuildFromScan is Rebuild fed by an existing user file scan, so startup
// can share one scan between the user index and the character index instead
// of fully parsing every user record a second time.
func (ci *CharacterIndex) RebuildFromScan(scan []UserFileScan) {
	newMap := make(map[string]int, len(scan))

	for _, s := range scan {
		if s.CharacterName != "" {
			newMap[strings.ToLower(s.CharacterName)] = s.UserId
		}
	}

	for _, u := range GetAllActiveUsers() {
		if u.Character != nil && u.Character.Name != "" {
			newMap[strings.ToLower(u.Character.Name)] = u.UserId
		}
	}

	ci.mu.Lock()
	ci.byName = newMap
	ci.mu.Unlock()
}
