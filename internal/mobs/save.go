package mobs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/casing"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/species"
)

// mobsDataRoot is the configured data-file root. It is a package var (rather
// than reading configs.GetFilePathsConfig() inline everywhere) so tests can
// redirect it at a temp dir without going through configs.SetVal, which
// persists to the real config-overrides.yaml on disk (see
// configs.overridePath) — a footgun for a unit test that must never touch the
// developer's real config. Mirrors items.itemsBasePath's test-injection shape.
var mobsDataRoot = func() string {
	return configs.GetFilePathsConfig().DataFiles.String()
}

// mobsBasePath is the root SaveFlatFile joins with Mob.Filepath().
func mobsBasePath() string {
	return mobsDataRoot() + `/mobs`
}

// CanonicalizeMobName normalizes the mob's player-visible name to the Title
// casing LoadDataFiles' casing.AssertCanonical demands at boot — a single
// non-canonical save (or an unedited "new mob" stub) bricks the NEXT boot.
// Every write path must run this first. Unlike items.CanonicalizeItemNames,
// there is no color-tag guard here: mob names are never color-tagged (a
// tagged mob name would already fail casing.AssertCanonical at boot today),
// so unconditionally Title()-ing is safe.
func CanonicalizeMobName(m *Mob) {
	m.Character.Name = casing.Title(m.Character.Name)
}

// AllScheduleIds returns every loaded schedule id, sorted, for the builder's
// dropdowns and for ValidateMobSpec's error messages.
func AllScheduleIds() []string {
	schedulesMu.RLock()
	defer schedulesMu.RUnlock()
	out := make([]string, 0, len(schedules))
	for id := range schedules {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// AllPatrolIds returns every loaded patrol id, sorted, for the builder's
// dropdowns and for ValidateMobSpec's error messages.
func AllPatrolIds() []string {
	patrolsMu.RLock()
	defer patrolsMu.RUnlock()
	out := make([]string, 0, len(patrols))
	for id := range patrols {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// isInternedDescriptionToken reports whether desc is a CacheDescription
// product (`h:` + 64 hex chars) rather than authored prose. Deliberately
// narrow so prose that happens to start with "h:" is never refused.
func isInternedDescriptionToken(desc string) bool {
	rest, ok := strings.CutPrefix(desc, `h:`)
	if !ok || len(rest) != 64 {
		return false
	}
	for _, r := range rest {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

var validArchetypes = map[string]bool{"": true, "fighting": true, "casting": true}

var validAIProfiles = map[string]bool{
	"": true, "default": true, "aggressive": true, "defensive": true,
	"grappler": true, "brawler": true, "tactical": true,
}

var validSubmissionPolicies = map[string]bool{
	"": true, "mercy": true, "subdue": true, "cripple": true, "lethal": true,
}

// ValidateMobSpec replicates every boot-time check the mobs package can reach
// on its own (the gmcp builder layer adds zone/craft-support/recipe checks
// that would otherwise create an import cycle). Errors name the offending
// field and list valid values so the web builder can surface them verbatim.
func ValidateMobSpec(m *Mob) error {
	if strings.TrimSpace(m.Character.Name) == "" {
		return fmt.Errorf("name is required")
	}
	// The boot path interns every template description into an `h:<sha256>`
	// token (characters.CacheDescription). Persisting a token destroys the
	// prose: the next boot interns the token string itself and the original
	// text is unrecoverable. Callers must resolve tokens back to prose
	// (characters.ResolveDescriptionToken) before saving.
	if isInternedDescriptionToken(m.Character.Description) {
		return fmt.Errorf("description is an interned h:<hash> token, not prose — resolve it before saving")
	}
	// Zone is intentionally NOT required: a mob template can be authored with
	// no home zone (a summon-only kind, or a new-mob stub not yet placed
	// anywhere). Filepath() routes an empty zone to the fixed "unzoned"
	// folder, so an empty zone is a fully boot-safe, loadable state.
	if m.ScheduleId != "" && GetSchedule(m.ScheduleId) == nil {
		return fmt.Errorf("schedule_id %q does not resolve; valid: %s", m.ScheduleId, strings.Join(AllScheduleIds(), ", "))
	}
	if m.PatrolId != "" && GetPatrol(m.PatrolId) == nil {
		return fmt.Errorf("patrol_id %q does not resolve; valid: %s", m.PatrolId, strings.Join(AllPatrolIds(), ", "))
	}
	if m.BehaviorArchetype != "" {
		p := filepath.Join(mobsDataRoot(), `behaviors`, `archetypes`, m.BehaviorArchetype+`.yaml`)
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("behavior_archetype %q has no archetype file at %s", m.BehaviorArchetype, p)
		}
	}
	if species.GetSpecies(m.Character.SpeciesId) == nil {
		return fmt.Errorf("speciesid %d is not a valid species", m.Character.SpeciesId)
	}
	if !validArchetypes[m.Archetype] {
		return fmt.Errorf(`archetype %q invalid; valid: "", fighting, casting`, m.Archetype)
	}
	if !validAIProfiles[m.AIProfile] {
		return fmt.Errorf(`aiprofile %q invalid; valid: "", default, aggressive, defensive, grappler, brawler, tactical`, m.AIProfile)
	}
	if !validSubmissionPolicies[m.SubmissionPolicy] {
		return fmt.Errorf(`submission_policy %q invalid; valid: "", mercy, subdue, cripple, lethal`, m.SubmissionPolicy)
	}
	// surrender_policy: "", "never", "always", or "auto-tap-below <N>"
	if sp := m.SurrenderPolicy; sp != "" && sp != "never" && sp != "always" && !strings.HasPrefix(sp, "auto-tap-below ") {
		return fmt.Errorf(`surrender_policy %q invalid; valid: "", never, always, "auto-tap-below <N>"`, sp)
	}
	for _, bid := range m.BuffIds {
		if buffs.GetBuffSpec(bid) == nil {
			return fmt.Errorf("buff id %d does not exist", bid)
		}
	}
	for _, iid := range m.LootPool {
		if items.GetItemSpec(iid) == nil {
			return fmt.Errorf("loot_pool item %d does not exist", iid)
		}
	}
	for _, si := range m.Character.Shop {
		if si.ItemId > 0 && items.GetItemSpec(si.ItemId) == nil {
			return fmt.Errorf("shop item %d does not exist", si.ItemId)
		}
	}
	for _, it := range m.Character.Items {
		if it.ItemId > 0 && items.GetItemSpec(it.ItemId) == nil {
			return fmt.Errorf("carried item %d does not exist", it.ItemId)
		}
	}
	for _, eq := range equippedItemIds(m.Character.Equipment) {
		if items.GetItemSpec(eq) == nil {
			return fmt.Errorf("equipped item %d does not exist", eq)
		}
	}
	if m.StatPool < 0 {
		return fmt.Errorf("statpool must be >= 0")
	}
	for _, check := range []struct {
		name string
		val  int
	}{
		{"activitylevel", m.ActivityLevel},
		{"itemdropchance", m.ItemDropChance},
		{"mutationchance", m.MutationChance},
		{"specialmovechance", m.SpecialMoveChance},
	} {
		if check.val < 0 || check.val > 100 {
			return fmt.Errorf("%s must be 0-100, got %d", check.name, check.val)
		}
	}
	return nil
}

// equippedItemIds collects the non-zero item ids across every Worn slot.
// Delegates to Worn.AllSlots() (the package's single source of truth for the
// slot list — see worn.go) instead of hand-enumerating fields, so a slot
// added later is picked up automatically instead of silently going
// unvalidated.
func equippedItemIds(w characters.Worn) []int {
	out := []int{}
	for _, s := range w.AllSlots() {
		if s.Item.ItemId > 0 {
			out = append(out, s.Item.ItemId)
		}
	}
	return out
}

// SaveMobSpec validates and writes a mob template, updating the in-memory
// cache. Mob filenames embed the (name-cached) canonical name under the zone
// folder, so a rename or re-zone changes the path. Filename()/Filepath() read
// mobNameCache first — so the OLD path MUST be computed from the template
// map (locked) before mobNameCache is updated, or "old" and "new" would
// already be identical. A stale duplicate must never linger to boot as a
// second copy of the mob.
//
// Invariant: a failed save must leave the caches consistent with disk (and
// with the mobs map, which is only updated after a successful write). The
// items seam (SaveItemSpec) gets this for free by writing to disk BEFORE
// touching its cache; this function can't use that ordering because
// fileloader.SaveFlatFile calls m.Filepath(), which reads mobNameCache — the
// NEW name must already be in the cache at write time. So the name-cache
// write happens first and is rolled back (via restoreCache) on any later
// failure, rather than reordered.
func SaveMobSpec(m Mob) error {
	CanonicalizeMobName(&m)
	if err := ValidateMobSpec(&m); err != nil {
		return err
	}
	base := mobsBasePath()

	mobsMu.RLock()
	old, existed := mobs[int(m.MobId)]
	mobsMu.RUnlock()
	var oldRel string
	if existed {
		oldRel = old.Filepath()
	}

	// Update the name cache BEFORE computing the new path — Filename() reads
	// mobNameCache, so the new path must be computed against the new name.
	// Capture the prior state so a later failure can roll this back.
	mobNameCacheMu.Lock()
	prevName, hadPrev := mobNameCache[m.MobId]
	mobNameCache[m.MobId] = m.Character.Name
	mobNameCacheMu.Unlock()

	restoreCache := func() {
		mobNameCacheMu.Lock()
		if hadPrev {
			mobNameCache[m.MobId] = prevName
		} else {
			delete(mobNameCache, m.MobId)
		}
		mobNameCacheMu.Unlock()
	}

	newRel := m.Filepath()
	if existed && oldRel != newRel {
		if err := os.Remove(filepath.Join(base, filepath.FromSlash(oldRel))); err != nil && !os.IsNotExist(err) {
			restoreCache()
			return err
		}
	}

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}
	if err := fileloader.SaveFlatFile[*Mob](base, &m, saveModes...); err != nil {
		restoreCache()
		return err
	}

	cp := m
	mobsMu.Lock()
	mobs[int(m.MobId)] = &cp
	mobsMu.Unlock()
	return nil
}

// DeleteMobSpec removes a mob template's file and both cache entries. Callers
// must first confirm the mob is unreferenced (the web builder's reference
// scan) — this function does not check.
func DeleteMobSpec(mobId MobId) error {
	mobsMu.RLock()
	tmpl, ok := mobs[int(mobId)]
	mobsMu.RUnlock()
	if !ok {
		return fmt.Errorf(`mob %d not found`, int(mobId))
	}
	if err := os.Remove(filepath.Join(mobsBasePath(), filepath.FromSlash(tmpl.Filepath()))); err != nil && !os.IsNotExist(err) {
		return err
	}
	mobsMu.Lock()
	delete(mobs, int(mobId))
	mobsMu.Unlock()
	mobNameCacheMu.Lock()
	delete(mobNameCache, mobId)
	mobNameCacheMu.Unlock()
	return nil
}

// CreateNewMobFile seeds a boot-safe stub mob in the given zone (or with no
// zone at all, for a summon-only / not-yet-placed template — see
// (*Mob).Filepath) at the next free mob id (template-cache max + 1 — the
// cache mirrors the filesystem at boot and every builder save keeps it
// fresh) and persists it via SaveMobSpec.
func CreateNewMobFile(zone string) (MobId, error) {
	mobsMu.RLock()
	nextId := 0
	for id := range mobs {
		if id > nextId {
			nextId = id
		}
	}
	mobsMu.RUnlock()
	nextId++

	stub := Mob{
		MobId:         MobId(nextId),
		Zone:          zone,
		StatPool:      10,
		ActivityLevel: 5,
	}
	stub.Character = characters.Character{
		Name:        fmt.Sprintf("New Mob %d", nextId),
		Description: "An unfinished mob.",
		SpeciesId:   1,
	}

	if err := SaveMobSpec(stub); err != nil {
		return 0, err
	}
	return MobId(nextId), nil
}
