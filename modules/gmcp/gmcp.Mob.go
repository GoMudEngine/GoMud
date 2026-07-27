package gmcp

// Build.Mob.* GMCP packages — the server side of the admin web mob-template
// editor (admin web-building 3). Mirrors gmcp.Item.go: RoleAdmin-gated (at
// dispatch, in gmcp.go / handleBuildOp), deferred to MainWorker via
// GMCPBuildOp (mob templates live in a shared package map + on disk, so
// mutating them off the connection goroutine would race the world tick).
// The mutating core funcs (buildMobList/Get/Create/Update) take a mobDeps
// seam in and return a BuildResult, so they're unit-testable against a fake
// world for load/save/create/zone-check behavior. buildMobGet's attached
// Enums, however, come from collectMobEnums(), which reads the mobs/species/
// rooms/shops package globals directly rather than through mobDeps — that
// data is read-only reference/dropdown content (valid archetypes, species,
// zones, schedule/patrol ids...), not world state a test needs to fake, so it
// is intentionally un-injected. A test asserting on Enums exercises the real
// registries, not a fake.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/llm"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ---- client -> server payloads ----

type mobGetReq struct {
	MobId int `json:"mobId"`
}
type mobCreateReq struct {
	Zone string `json:"zone"`
}
type mobDeleteReq struct {
	MobId int `json:"mobId"`
}
type mobSpawnReq struct {
	MobId  int `json:"mobId"`
	RoomId int `json:"roomId"`
}

type mobShopRow struct {
	ItemId      int `json:"itemId"`
	QuantityMax int `json:"quantityMax"`
	Price       int `json:"price"`
}
type mobRelRow struct {
	To      int    `json:"to"`
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
}

// mobUpdateReq is both the Save payload and (embedded) the echoed detail. The
// full-parity field list is the contract with the web form — every field the
// character/mob YAML schema exposes to the builder lives here under a stable
// json name; the UI tasks (5-7) were planned against these exact names.
type mobUpdateReq struct {
	MobId int    `json:"mobId"`
	Zone  string `json:"zone"`
	// character block
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Adjectives   []string       `json:"adjectives"`
	SpeciesId    int            `json:"speciesId"`
	Gold         int            `json:"gold"`
	StatTraining map[string]int `json:"statTraining"` // strength/dexterity/perception/vitality/willpower/charisma
	Equipment    map[string]int `json:"equipment"`    // worn-slot key (characters.WornSlot.Key) -> itemId (0/absent = empty)
	CarriedItems []int          `json:"carriedItems"` // itemIds
	Shop         []mobShopRow   `json:"shop"`
	// stats & combat
	StatPool          int            `json:"statPool"`
	Archetype         string         `json:"archetype"`
	AutoAggro         bool           `json:"autoAggro"`
	AIProfile         string         `json:"aiProfile"`
	SpecialMoveChance int            `json:"specialMoveChance"`
	MovePreferences   map[string]int `json:"movePreferences"`
	ActivityLevel     int            `json:"activityLevel"`
	MaxWander         int            `json:"maxWander"`
	Routine           string         `json:"routine"`
	RoutineLinks      []string       `json:"routineLinks"`
	Hates             []string       `json:"hates"`
	Groups            []string       `json:"groups"`
	SubmissionPolicy  string         `json:"submissionPolicy"`
	SurrenderPolicy   string         `json:"surrenderPolicy"`
	PackFleeImmune    bool           `json:"packFleeImmune"`
	// flavor
	IdleCommands   []string `json:"idleCommands"`
	CombatCommands []string `json:"combatCommands"`
	AngryCommands  []string `json:"angryCommands"`
	// loot
	ItemDropChance int   `json:"itemDropChance"`
	LootPool       []int `json:"lootPool"`
	// commerce
	BuysGeneral             bool     `json:"buysGeneral"`
	StockMultiplier         float64  `json:"stockMultiplier"`
	CraftSupport            string   `json:"craftSupport"`
	Crafter                 bool     `json:"crafter"`
	CrafterSkill            string   `json:"crafterSkill"`
	CrafterRecipeIds        []string `json:"crafterRecipeIds"`
	CrafterRestockMaterials []int    `json:"crafterRestockMaterials"`
	// aliveness
	ScheduleId         string      `json:"scheduleId"`
	PatrolId           string      `json:"patrolId"`
	Relationships      []mobRelRow `json:"relationships"`
	KnowsFacts         []string    `json:"knowsFacts"`
	DefaultDisposition int         `json:"defaultDisposition"`
	FoldAnchorRoom     int         `json:"foldAnchorRoom"`
	StorageChestRoom   int         `json:"storageChestRoom"`
	// hooks
	ScriptTag         string   `json:"scriptTag"`
	BehaviorArchetype string   `json:"behaviorArchetype"`
	BuffIds           []int    `json:"buffIds"`
	QuestFlags        []string `json:"questFlags"`
	SpawnMutations    []string `json:"spawnMutations"`
	MutationChance    int      `json:"mutationChance"`
	LLMProfileJSON    string   `json:"llmProfileJson"` // raw round-trip; "" = clear, "-" sentinel = untouched (Update route only)
	// overrides & flags
	CarryCapacity      float64 `json:"carryCapacity"`
	HealthMax          int     `json:"healthMax"`
	StaminaMax         int     `json:"staminaMax"`
	CorpseName         string  `json:"corpseName"`
	CorpseDescription  string  `json:"corpseDescription"`
	HideEquipmentSlots bool    `json:"hideEquipmentSlots"`
	CharmImmune        bool    `json:"charmImmune"`
	NonCombatant       bool    `json:"nonCombatant"`
	PlayerAttackImmune bool    `json:"playerAttackImmune"`
}

// ---- server -> client detail (Build.Mob) ----

type mobEnums struct {
	Zones              []string          `json:"zones"`
	Species            map[string]string `json:"species"` // id (string key, for JSON) -> name
	Archetypes         []string          `json:"archetypes"`
	AIProfiles         []string          `json:"aiProfiles"`
	BehaviorArchetypes []string          `json:"behaviorArchetypes"`
	ScheduleIds        []string          `json:"scheduleIds"`
	PatrolIds          []string          `json:"patrolIds"`
	CraftSupports      []string          `json:"craftSupports"`
	CrafterSkills      []string          `json:"crafterSkills"` // recipe disciplines; excludes "general"
	SubmissionPolicies []string          `json:"submissionPolicies"`
	WornSlots          []string          `json:"wornSlots"`
	Groups             []string          `json:"groups"` // observed values across existing mobs, as suggestions
	Buffs              []idName          `json:"buffs"`  // id pickers (epic followup: no more bare-numeric buff ids)
}

type mobDetail struct {
	mobUpdateReq
	Enums mobEnums `json:"enums"`
	// HasDialogue drives the mob form's Dialogue button label (5b).
	HasDialogue bool `json:"hasDialogue"`
}

// mobListRow is one row of Build.Mobs.
type mobListRow struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Zone         string `json:"zone"`
	StatPool     int    `json:"statPool"`
	NonCombatant bool   `json:"nonCombatant"`
	HasSchedule  bool   `json:"hasSchedule"`
	HasShop      bool   `json:"hasShop"`
}

// ---- dependency seam ----

// mobRefEntry describes one live reference to a mob template, surfaced by
// Build.Mob.Delete when deletion is blocked. Named distinctly from the
// package-level `mobRef` already defined in gmcp.Item_refs.go (an unrelated
// internal type used while scanning items FOR referencing mobs) to avoid a
// symbol collision within the package.
type mobRefEntry struct {
	Kind string `json:"kind"` // room-spawn | relationship | quest | dialogue | conversation | merc-shop
	Id   string `json:"id"`
}

type mobDeps struct {
	load       func(id int) *mobs.Mob
	save       func(m mobs.Mob) error
	del        func(id int) error
	create     func(zone string) (int, error)
	zoneExists func(zone string) bool
	// references is wired to scanMobReferences (below) by realMobDeps(); spawn
	// is wired to spawnMobInRoom (below) — the test-spawn control (Task 4).
	references func(id int) []mobRefEntry
	spawn      func(mobId, roomId int) (string, error)
}

func realMobDeps() mobDeps {
	return mobDeps{
		load: func(id int) *mobs.Mob { return mobs.GetMobSpec(mobs.MobId(id)) },
		save: mobs.SaveMobSpec,
		del:  func(id int) error { return mobs.DeleteMobSpec(mobs.MobId(id)) },
		create: func(zone string) (int, error) {
			id, err := mobs.CreateNewMobFile(zone)
			return int(id), err
		},
		zoneExists: func(zone string) bool { return rooms.GetZoneConfig(zone) != nil },
		references: scanMobReferences,
		spawn:      spawnMobInRoom,
	}
}

// ---- core ----

func buildMobList(d mobDeps) []mobListRow {
	rows := []mobListRow{}
	for _, m := range mobs.AllMobTemplates() {
		rows = append(rows, mobListRow{
			Id: int(m.MobId), Name: m.Character.Name, Zone: m.Zone, StatPool: m.StatPool,
			NonCombatant: m.IsNonCombatant(), HasSchedule: m.ScheduleId != "", HasShop: len(m.Character.Shop) > 0,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Id < rows[j].Id })
	return rows
}

func mobToReq(m *mobs.Mob) mobUpdateReq {
	// Templates carry an interned `h:<hash>` description after boot
	// (characters.CacheDescription); the editor gets the prose. An
	// unresolvable token falls through as-is — buildMobUpdate refuses it on
	// the way back, so it can't reach a YAML file.
	desc, ok := characters.ResolveDescriptionToken(m.Character.Description)
	if !ok {
		desc = m.Character.Description
	}
	req := mobUpdateReq{
		MobId: int(m.MobId), Zone: m.Zone,
		Name: m.Character.Name, Description: desc, Adjectives: m.Character.Adjectives,
		SpeciesId: m.Character.SpeciesId, Gold: m.Character.Gold,
		StatPool: m.StatPool, Archetype: m.Archetype, AutoAggro: m.AutoAggro, AIProfile: m.AIProfile,
		SpecialMoveChance: m.SpecialMoveChance, MovePreferences: m.MovePreferences,
		ActivityLevel: m.ActivityLevel, MaxWander: m.MaxWander,
		Routine: m.Routine, RoutineLinks: m.RoutineLinks, Hates: m.Hates, Groups: m.Groups,
		SubmissionPolicy: m.SubmissionPolicy, SurrenderPolicy: m.SurrenderPolicy, PackFleeImmune: m.PackFleeImmune,
		IdleCommands: m.IdleCommands, CombatCommands: m.CombatCommands, AngryCommands: m.AngryCommands,
		ItemDropChance: m.ItemDropChance, LootPool: m.LootPool,
		BuysGeneral: m.BuysGeneral, StockMultiplier: m.StockMultiplier, CraftSupport: m.ShopCraftSupport,
		Crafter: m.Crafter, CrafterSkill: m.CrafterSkill, CrafterRecipeIds: m.CrafterRecipeIds,
		CrafterRestockMaterials: m.CrafterRestockMaterials,
		ScheduleId:              m.ScheduleId, PatrolId: m.PatrolId,
		KnowsFacts: m.KnowsFacts, DefaultDisposition: m.DefaultDisposition,
		FoldAnchorRoom: m.FoldAnchorRoom, StorageChestRoom: m.StorageChestRoom,
		ScriptTag: m.ScriptTag, BehaviorArchetype: m.BehaviorArchetype,
		BuffIds: m.BuffIds, QuestFlags: m.QuestFlags,
		SpawnMutations: m.SpawnMutations, MutationChance: m.MutationChance,
		CarryCapacity: m.CarryCapacityOverride, HealthMax: m.HealthMaxOverride, StaminaMax: m.StaminaMaxOverride,
		CorpseName: m.CorpseName, CorpseDescription: m.CorpseDescription,
		HideEquipmentSlots: m.HideEquipmentSlots, CharmImmune: m.CharmImmune,
		NonCombatant: m.NonCombatant, PlayerAttackImmune: m.PlayerAttackImmune,
	}
	req.StatTraining = map[string]int{
		"strength": m.Character.Stats.Strength.Training, "dexterity": m.Character.Stats.Dexterity.Training,
		"perception": m.Character.Stats.Perception.Training, "vitality": m.Character.Stats.Vitality.Training,
		"willpower": m.Character.Stats.Willpower.Training, "charisma": m.Character.Stats.Charisma.Training,
	}
	req.Equipment = wornToMap(m.Character.Equipment)
	for _, it := range m.Character.Items {
		req.CarriedItems = append(req.CarriedItems, it.ItemId)
	}
	for _, si := range m.Character.Shop {
		if si.ItemId > 0 {
			req.Shop = append(req.Shop, mobShopRow{ItemId: si.ItemId, QuantityMax: si.QuantityMax, Price: si.Price})
		}
	}
	for _, r := range m.Relationships {
		req.Relationships = append(req.Relationships, mobRelRow{To: r.To, Type: r.Type, Subtype: r.Subtype})
	}
	req.LLMProfileJSON = llmProfileToJSON(m.LLMProfile) // "" when nil
	return req
}

func buildMobGet(d mobDeps, mobId int) (mobDetail, bool) {
	m := d.load(mobId)
	if m == nil {
		return mobDetail{}, false
	}
	// dialogue.Load is NOT side-effect-free: for a dialogueless mob it sets
	// the nil sentinel. That is correct — CreateNewDialogueFile clears it —
	// but do not "optimize" this call thinking it is a pure read.
	hasDlg := dialogue.Load(mobId, m.Zone) != nil
	return mobDetail{mobUpdateReq: mobToReq(m), Enums: collectMobEnums(), HasDialogue: hasDlg}, true
}

// reqToMob starts from a copy of the loaded template so anything the form
// doesn't carry (BTreeState, VisitedZones, GivenItems, InstanceId, ...)
// survives a save untouched. Relationships/KnowsFacts are special-cased: a
// nil slice in the request means the form didn't touch that section (leave
// the base's value alone), while every other slice field is replaced
// wholesale — matching the form's "always sends its full state" contract for
// everything except those two sections, which the mob form doesn't expose a
// full editor for yet.
func reqToMob(base *mobs.Mob, req mobUpdateReq) mobs.Mob {
	m := *base
	m.Zone = req.Zone
	m.Character.Name, m.Character.Description = req.Name, req.Description
	m.Character.Adjectives, m.Character.SpeciesId, m.Character.Gold = req.Adjectives, req.SpeciesId, req.Gold
	// The form always submits all six stat keys together — but apply each
	// key only if present, rather than unconditionally reading the map (which
	// would silently zero a stat's Training on any partial/malformed payload
	// instead of leaving the base value alone).
	if v, ok := req.StatTraining["strength"]; ok {
		m.Character.Stats.Strength.Training = v
	}
	if v, ok := req.StatTraining["dexterity"]; ok {
		m.Character.Stats.Dexterity.Training = v
	}
	if v, ok := req.StatTraining["perception"]; ok {
		m.Character.Stats.Perception.Training = v
	}
	if v, ok := req.StatTraining["vitality"]; ok {
		m.Character.Stats.Vitality.Training = v
	}
	if v, ok := req.StatTraining["willpower"]; ok {
		m.Character.Stats.Willpower.Training = v
	}
	if v, ok := req.StatTraining["charisma"]; ok {
		m.Character.Stats.Charisma.Training = v
	}
	// Equipment/carried items are base-aware: an id that matches what the base
	// template already had in that slot (or already held in inventory) keeps
	// the EXISTING item instance — preserving EnchantType/EnchantTier,
	// Adjectives, non-default Uses, Blob, and any Spec override authored on
	// that specific instance. items.New(id) (a bare spec-default instance) is
	// only used for an id that is new or changed in that slot/list. Without
	// this, saving any unrelated field on a mob with an enchanted boss weapon
	// would silently revert that weapon to a spec-default instance.
	m.Character.Equipment = mapToWornPreserving(base.Character.Equipment, req.Equipment)
	m.Character.Items = mergeCarriedItems(base.Character.Items, req.CarriedItems)
	m.Character.Shop = nil
	for _, row := range req.Shop {
		if row.ItemId > 0 {
			m.Character.Shop = append(m.Character.Shop, characters.ShopItem{ItemId: row.ItemId, QuantityMax: row.QuantityMax, Price: row.Price})
		}
	}
	m.StatPool, m.Archetype, m.AutoAggro, m.AIProfile = req.StatPool, req.Archetype, req.AutoAggro, req.AIProfile
	m.LegacyHostile = false // auto_aggro (above) is the canonical field now; a save migrates any legacy `hostile:` file
	m.SpecialMoveChance, m.MovePreferences = req.SpecialMoveChance, req.MovePreferences
	m.ActivityLevel, m.MaxWander = req.ActivityLevel, req.MaxWander
	m.Routine, m.RoutineLinks, m.Hates, m.Groups = req.Routine, req.RoutineLinks, req.Hates, req.Groups
	m.SubmissionPolicy, m.SurrenderPolicy, m.PackFleeImmune = req.SubmissionPolicy, req.SurrenderPolicy, req.PackFleeImmune
	m.IdleCommands, m.CombatCommands, m.AngryCommands = req.IdleCommands, req.CombatCommands, req.AngryCommands
	m.ItemDropChance, m.LootPool = req.ItemDropChance, req.LootPool
	m.BuysGeneral, m.StockMultiplier, m.ShopCraftSupport = req.BuysGeneral, req.StockMultiplier, req.CraftSupport
	m.Crafter, m.CrafterSkill = req.Crafter, req.CrafterSkill
	m.CrafterRecipeIds, m.CrafterRestockMaterials = req.CrafterRecipeIds, req.CrafterRestockMaterials
	m.ScheduleId, m.PatrolId = req.ScheduleId, req.PatrolId
	if req.Relationships != nil {
		m.Relationships = nil
		for _, r := range req.Relationships {
			m.Relationships = append(m.Relationships, mobs.RelationshipYAMLEntry{To: r.To, Type: r.Type, Subtype: r.Subtype})
		}
	}
	if req.KnowsFacts != nil {
		m.KnowsFacts = req.KnowsFacts
	}
	m.DefaultDisposition, m.FoldAnchorRoom, m.StorageChestRoom = req.DefaultDisposition, req.FoldAnchorRoom, req.StorageChestRoom
	m.ScriptTag, m.BehaviorArchetype = req.ScriptTag, req.BehaviorArchetype
	m.BuffIds, m.QuestFlags = req.BuffIds, req.QuestFlags
	m.SpawnMutations, m.MutationChance = req.SpawnMutations, req.MutationChance
	if req.LLMProfileJSON != "-" {
		// "-" is the Update route's sentinel meaning "field absent from this
		// payload, leave whatever the base template already has." Anything
		// else (including "") is an explicit value: "" clears the profile.
		// buildMobUpdate has already validated the JSON (surfacing a parse
		// error to the caller before reaching here), so the error is safe to
		// discard on this second, purely mechanical parse.
		profile, _ := llmProfileFromJSON(req.LLMProfileJSON)
		m.LLMProfile = profile
	}
	m.CarryCapacityOverride, m.HealthMaxOverride, m.StaminaMaxOverride = req.CarryCapacity, req.HealthMax, req.StaminaMax
	m.CorpseName, m.CorpseDescription = req.CorpseName, req.CorpseDescription
	m.HideEquipmentSlots, m.CharmImmune = req.HideEquipmentSlots, req.CharmImmune
	m.NonCombatant, m.PlayerAttackImmune = req.NonCombatant, req.PlayerAttackImmune
	return m
}

// buildMobUpdate runs the gmcp-layer checks the mobs package can't reach on
// its own (zone existence, craft-support, recipe ids — packages that would
// create an import cycle if internal/mobs imported them), parses the raw
// LLMProfileJSON up front (so a malformed payload is rejected with a clear
// error instead of silently discarding the profile), and then saves — which
// runs mobs.ValidateMobSpec for everything that package CAN reach.
func buildMobUpdate(d mobDeps, req mobUpdateReq) BuildResult {
	base := d.load(req.MobId)
	if base == nil {
		return buildErr("mob %d not found", req.MobId)
	}
	if req.Name == "" {
		return buildErr("name is required")
	}
	// A stale client can echo the interned `h:<hash>` description token back
	// unedited. Resolve it to the prose; refuse an unresolvable token —
	// persisting one would destroy the description on the next boot.
	if resolved, ok := characters.ResolveDescriptionToken(req.Description); ok {
		req.Description = resolved
	} else {
		return buildErr("description is an unresolvable interned token (%.20s…) — reload the builder page and retype the description", req.Description)
	}
	// Empty zone is legal (a summon-only / not-yet-placed template) and is
	// NOT run through zoneExists — only a non-empty, UNKNOWN zone is rejected.
	if req.Zone != "" && !d.zoneExists(req.Zone) {
		return buildErr("zone %q does not exist", req.Zone)
	}
	if req.CraftSupport != "" && !shops.IsValidCraftSupport(req.CraftSupport) {
		return buildErr("craft_support %q invalid; valid: %s", req.CraftSupport, strings.Join(shops.ValidCraftSupports, ", "))
	}
	// CrafterSkill is matched verbatim against recipe.Skill (mobs.pickEligibleRecipe)
	// and passed to shops.EvaluateBuyRules, so a typo fails SILENTLY — the mob just
	// never crafts and its buy rules stop matching. Validate against the discipline
	// set (ValidVendorCategories), NOT ValidCraftSupports: "general" is a legal shop
	// support but is never a recipe skill, so a "general" crafter matches nothing.
	if req.CrafterSkill != "" && !shops.IsValidVendorCategory(req.CrafterSkill) {
		return buildErr("crafter_skill %q invalid; valid: %s", req.CrafterSkill, strings.Join(shops.ValidVendorCategories, ", "))
	}
	if len(req.CrafterRecipeIds) > 0 {
		all := crafting.GetAll()
		for _, rid := range req.CrafterRecipeIds {
			if _, ok := all[rid]; !ok {
				return buildErr("crafter_recipe_id %q does not exist", rid)
			}
		}
	}
	if req.LLMProfileJSON != "-" {
		if _, err := llmProfileFromJSON(req.LLMProfileJSON); err != nil {
			return buildErr("llm profile JSON invalid: %s", err.Error())
		}
	}
	if err := d.save(reqToMob(base, req)); err != nil {
		return buildErr("could not save mob %d: %s", req.MobId, err.Error())
	}
	return BuildResult{Ok: true, MobId: req.MobId}
}

func buildMobCreate(d mobDeps, zone string) BuildResult {
	// Empty zone is legal (a summon-only / not-yet-placed template) and skips
	// the zoneExists check entirely — only a non-empty, UNKNOWN zone is
	// rejected.
	if zone != "" && !d.zoneExists(zone) {
		return buildErr("zone %q does not exist", zone)
	}
	id, err := d.create(zone)
	if err != nil {
		return buildErr("%s", err.Error())
	}
	return BuildResult{Ok: true, MobId: id}
}

// buildMobDelete rejects deletion of a mob template that's still referenced
// anywhere in the world (the reference scan below), returning those
// references so the admin can go clean them up first, rather than leaving
// dangling room-spawns/relationships/dialogue/etc. pointing at a vanished
// template.
func buildMobDelete(d mobDeps, mobId int) BuildResult {
	if d.load(mobId) == nil {
		return buildErr("mob %d not found", mobId)
	}
	if refs := d.references(mobId); len(refs) > 0 {
		return BuildResult{Error: "mob is still referenced — remove these first", MobRefs: refs}
	}
	if err := d.del(mobId); err != nil {
		return buildErr("could not delete mob %d: %s", mobId, err.Error())
	}
	return BuildResult{Ok: true, MobId: mobId}
}

// buildMobSpawn test-spawns a mob template into an admin-chosen room — the web
// equivalent of the in-game `mob spawn` admin command. Rejects an unknown
// template before touching the world; any spawn-time failure (bad room id,
// instantiation failure) surfaces as a non-Ok result naming the problem.
func buildMobSpawn(d mobDeps, req mobSpawnReq) BuildResult {
	if d.load(req.MobId) == nil {
		return buildErr("mob %d not found", req.MobId)
	}
	name, err := d.spawn(req.MobId, req.RoomId)
	if err != nil {
		return buildErr("could not spawn: %s", err.Error())
	}
	return BuildResult{Ok: true, MobId: req.MobId, Message: fmt.Sprintf("%s spawned in room %d", name, req.RoomId)}
}

// spawnMobInRoom mirrors the in-game `mob spawn` admin command
// (internal/usercommands/admin.mob.go:200-209): NewMobById + room.AddMob + a
// room announce. Build.Mob.Spawn is dispatched via GMCPBuildOp and processed
// by handleBuildOp on MainWorker, so mutating the shared room/mob world state
// here is safe (no connection-goroutine race).
func spawnMobInRoom(mobId, roomId int) (string, error) {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return "", fmt.Errorf("room %d not found", roomId)
	}
	mob := mobs.NewMobById(mobs.MobId(mobId), roomId)
	if mob == nil {
		return "", fmt.Errorf("mob %d could not be instantiated", mobId)
	}
	room.AddMob(mob.InstanceId)
	// excludeUserIds left empty (no player performed this in-room) — 0 isn't
	// needed since SendTextVisual only excludes uids present in r.GetPlayers().
	room.SendTextVisual(messaging.CategoryMobEmote,
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> appears in the air and falls to the ground.`, mob.Character.Name))
	return mob.Character.Name, nil
}

// ---- reference scan (Build.Mob.Delete) ----
//
// mobRefIterators is the injected-iterator seam behind scanMobReferencesWith
// so the pure matching logic (does mobId appear in this candidate's id list)
// is unit-testable without touching the rooms/mobs/questengine/conversations
// package globals. scanMobReferences (below) wires it to the real world.
type mobRefIterators struct {
	roomSpawns       func(yield func(roomId int, mobIds []int))
	mobRelationships func(yield func(fromMobId int, name string, toIds []int))
	quests           func(yield func(questToken string, mobIds []int))
	convPairs        func(yield func(pairFile string, mobIds []int))
	mercShops        func(yield func(sellerMobId int, name string, mobIds []int))
}

func scanMobReferencesWith(mobId int, it mobRefIterators) []mobRefEntry {
	out := []mobRefEntry{}
	if it.roomSpawns != nil {
		it.roomSpawns(func(roomId int, ids []int) {
			if containsInt(ids, mobId) {
				out = append(out, mobRefEntry{Kind: "room-spawn", Id: fmt.Sprintf("room %d", roomId)})
			}
		})
	}
	if it.mobRelationships != nil {
		it.mobRelationships(func(fromId int, name string, ids []int) {
			if containsInt(ids, mobId) {
				out = append(out, mobRefEntry{Kind: "relationship", Id: fmt.Sprintf("mob %d (%s)", fromId, name)})
			}
		})
	}
	if it.quests != nil {
		it.quests(func(token string, ids []int) {
			if containsInt(ids, mobId) {
				out = append(out, mobRefEntry{Kind: "quest", Id: "quest " + token})
			}
		})
	}
	if it.convPairs != nil {
		it.convPairs(func(file string, ids []int) {
			if containsInt(ids, mobId) {
				out = append(out, mobRefEntry{Kind: "conversation", Id: file})
			}
		})
	}
	if it.mercShops != nil {
		it.mercShops(func(sellerId int, name string, ids []int) {
			if containsInt(ids, mobId) {
				out = append(out, mobRefEntry{Kind: "merc-shop", Id: fmt.Sprintf("mob %d (%s) sells it", sellerId, name)})
			}
		})
	}
	return out
}

// scanMobReferences wires scanMobReferencesWith to the real world:
//
//   - roomSpawns: every loaded room's SpawnInfo entries (SpawnInfo.MobId).
//   - mobRelationships: every OTHER mob template's Relationships (RelationshipYAMLEntry.To).
//   - mercShops: every OTHER mob template's Character.Shop rows with MobId > 0
//     (mercenary-for-sale rows — see characters.ShopItem.MobId).
//   - quests: verified via codegraph — internal/quests.Quest/QuestStep carry NO
//     mob-id fields at all (that package is the legacy simple-reward quest
//     model: QuestId/Name/Description/Steps/Rewards/Flags, and QuestStep is
//     just Id/Description/Hint). The real quest CONTENT that names mob ids
//     lives in the separate internal/questengine package's expanded QuestDef
//     (loaded from _datafiles/world/dogmud/quests/*.yaml, registered on
//     questengine.GetEngine(), enumerable via (*Engine).AllQuests()).
//     QuestDef.Triggers ([]TriggerDef) carries mob ids in several places:
//     TriggerDef.Mob (the mob-death/etc. filter), ActionDef.NpcSay.Mob +
//     .Lines[].Speaker (which mob speaks), ActionDef.SpawnMob.Id (mob to
//     spawn — SpawnDef is shared with SpawnItem, so only SpawnMob is scanned,
//     never SpawnItem), and ActionDef.DeclareBounty.Target.Id when
//     Target.Type == "mob". All of these are collected per quest below.
//   - convPairs: verified via internal/conversations/loader.go + conversation.go
//     — pair overrides are loaded from {DataFiles}/conversations/pairs/*.yaml,
//     filename convention "<lowerMobId>_<higherMobId>.yaml" (loader.go enforces
//     this on load), registered in an unexported `pairs` map keyed by a
//     normalized pair, with no exported "list all" accessor — only
//     GetPairOverride(mobA, mobB) for a KNOWN pair. Since the ids to scan for
//     aren't known ahead of time, this reads the pairs directory directly and
//     parses both mob ids straight out of each filename (the same convention
//     the loader validates against), rather than going through the package.
//   - dialogue: a single file check (not an iterator) — dialogue files are
//     keyed "{DataFiles}/dialogue/{ZoneNameSanitize(zone)}/{mobId}.yaml"
//     (dialogue/loader.go:34) — appended after the generic iterators since it
//     needs the mob's OWN zone rather than scanning other mobs/rooms/quests.
func scanMobReferences(mobId int) []mobRefEntry {
	dataRoot := configs.GetFilePathsConfig().DataFiles.String()
	m := mobs.GetMobSpec(mobs.MobId(mobId))
	refs := scanMobReferencesWith(mobId, mobRefIterators{
		roomSpawns: func(yield func(int, []int)) {
			for _, roomId := range rooms.GetAllRoomIds() {
				r := rooms.LoadRoom(roomId)
				if r == nil {
					continue
				}
				ids := []int{}
				for _, si := range r.SpawnInfo {
					if si.MobId > 0 {
						ids = append(ids, si.MobId)
					}
				}
				if len(ids) > 0 {
					yield(roomId, ids)
				}
			}
		},
		mobRelationships: func(yield func(int, string, []int)) {
			for _, other := range mobs.AllMobTemplates() {
				if int(other.MobId) == mobId {
					continue
				}
				ids := []int{}
				for _, rel := range other.Relationships {
					ids = append(ids, rel.To)
				}
				if len(ids) > 0 {
					yield(int(other.MobId), other.Character.Name, ids)
				}
			}
		},
		quests: func(yield func(string, []int)) {
			for _, q := range questengine.GetEngine().AllQuests() {
				ids := []int{}
				for _, t := range q.Triggers {
					if t.Mob > 0 {
						ids = append(ids, t.Mob)
					}
					for _, a := range t.Actions {
						if a.NpcSay != nil {
							if a.NpcSay.Mob > 0 {
								ids = append(ids, a.NpcSay.Mob)
							}
							for _, line := range a.NpcSay.Lines {
								if line.Speaker > 0 {
									ids = append(ids, line.Speaker)
								}
							}
						}
						if a.SpawnMob != nil && a.SpawnMob.Id > 0 {
							ids = append(ids, a.SpawnMob.Id)
						}
						if a.DeclareBounty != nil && a.DeclareBounty.Target != nil &&
							a.DeclareBounty.Target.Type == "mob" && a.DeclareBounty.Target.Id > 0 {
							ids = append(ids, a.DeclareBounty.Target.Id)
						}
					}
				}
				if len(ids) > 0 {
					yield(fmt.Sprintf("%d (%s)", q.QuestId, q.Name), ids)
				}
			}
		},
		mercShops: func(yield func(int, string, []int)) {
			for _, other := range mobs.AllMobTemplates() {
				if int(other.MobId) == mobId {
					continue
				}
				ids := []int{}
				for _, si := range other.Character.Shop {
					if si.MobId > 0 {
						ids = append(ids, si.MobId)
					}
				}
				if len(ids) > 0 {
					yield(int(other.MobId), other.Character.Name, ids)
				}
			}
		},
		convPairs: func(yield func(string, []int)) {
			dir := util.FilePath(dataRoot, `/`, `conversations`, `/`, `pairs`)
			entries, err := os.ReadDir(dir)
			if err != nil {
				return // no pairs directory — nothing to scan (optional content)
			}
			for _, en := range entries {
				if en.IsDir() {
					continue
				}
				base, ok := strings.CutSuffix(en.Name(), ".yaml")
				if !ok {
					continue
				}
				parts := strings.SplitN(base, "_", 2)
				if len(parts) != 2 {
					continue
				}
				a, errA := strconv.Atoi(parts[0])
				b, errB := strconv.Atoi(parts[1])
				if errA != nil || errB != nil {
					continue
				}
				yield("conversations/pairs/"+en.Name(), []int{a, b})
			}
		},
	})
	// Dialogue file keyed by this mob's own id/zone — checked separately since
	// it needs the mob's own zone, not another mob/room/quest's data. Skipped
	// entirely for a zoneless mob: dialogue files are zone-keyed
	// ("dialogue/{zone}/{mobId}.yaml"), so a zoneless template can never have
	// one and the os.Stat would only add a pointless zero-zone lookup.
	if m != nil && m.Zone != "" {
		zone := mobs.ZoneNameSanitize(m.Zone)
		p := util.FilePath(dataRoot, `/`, `dialogue`, `/`, zone, `/`, strconv.Itoa(mobId)+`.yaml`)
		if _, err := os.Stat(p); err == nil {
			refs = append(refs, mobRefEntry{Kind: "dialogue", Id: fmt.Sprintf("dialogue/%s/%d.yaml", zone, mobId)})
		}
	}
	return refs
}

// ---- enum providers ----

func collectMobEnums() mobEnums {
	e := mobEnums{
		Archetypes:         []string{"", "fighting", "casting"},
		AIProfiles:         []string{"", "default", "aggressive", "defensive", "grappler", "brawler", "tactical"},
		ScheduleIds:        mobs.AllScheduleIds(),
		PatrolIds:          mobs.AllPatrolIds(),
		CraftSupports:      shops.ValidCraftSupports,
		CrafterSkills:      shops.ValidVendorCategories,
		SubmissionPolicies: []string{"", "mercy", "subdue", "cripple", "lethal"},
		WornSlots:          wornSlotNames(),
		Species:            map[string]string{},
	}
	for _, id := range buffs.GetAllBuffIds() {
		if spec := buffs.GetBuffSpec(id); spec != nil {
			e.Buffs = append(e.Buffs, idName{Id: id, Name: spec.Name})
		}
	}
	sort.Slice(e.Buffs, func(i, j int) bool { return e.Buffs[i].Id < e.Buffs[j].Id })
	for _, s := range species.GetAllSpecies() {
		e.Species[fmt.Sprintf("%d", s.SpeciesId)] = s.Name
	}
	e.Zones = allZoneNames()
	e.BehaviorArchetypes = listBehaviorArchetypeFiles()
	seen := map[string]bool{}
	for _, m := range mobs.AllMobTemplates() {
		for _, g := range m.Groups {
			if !seen[g] {
				seen[g] = true
				e.Groups = append(e.Groups, g)
			}
		}
	}
	sort.Strings(e.Groups)
	return e
}

// allZoneNames reuses the same zone list the room builder ships in its
// Zone.Map payload (rooms.GetAllZoneNames), so the mob editor's zone dropdown
// and the room editor's zone switcher never diverge.
func allZoneNames() []string {
	out := rooms.GetAllZoneNames()
	sort.Strings(out)
	return out
}

func listBehaviorArchetypeFiles() []string {
	dir := configs.GetFilePathsConfig().DataFiles.String() + `/behaviors/archetypes`
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := []string{}
	for _, en := range entries {
		if n, ok := strings.CutSuffix(en.Name(), ".yaml"); ok {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// ---- Worn slot mapping ----
// wornToMap / mapToWornPreserving / wornSlotNames all derive from
// characters.Worn's AllSlots() — the package's single source of truth for the
// equipment slot list (24 slots as of Extra Arms/Tail/ComponentBag: weapon,
// offhand, extraarm1-4, head, neck, shoulders, body, back, belt, wrist1/2,
// extrawrist1-4, gloves, ring, ring2, legs, feet, tail, componentbag).
// internal/mobs/save.go's equippedItemIds validator already iterates
// AllSlots() for the same reason: a slot added later is picked up here
// automatically instead of silently going unmapped by the builder.
func wornSlotNames() []string {
	var w characters.Worn
	out := make([]string, 0, 24)
	for _, s := range w.AllSlots() {
		out = append(out, s.Key)
	}
	return out
}

func wornToMap(w characters.Worn) map[string]int {
	out := map[string]int{}
	for _, s := range w.AllSlots() {
		if s.Item.ItemId > 0 {
			out[s.Key] = s.Item.ItemId
		}
	}
	return out
}

// mapToWornPreserving rebuilds a Worn from the submitted slot->itemId map,
// base-aware: when the submitted id for a slot matches what base already had
// equipped there, the EXISTING item instance is carried forward unchanged
// (preserving EnchantType/EnchantTier, Adjectives, non-default Uses, Blob,
// and any per-instance Spec override — all real yaml fields on items.Item
// that a bare items.New(id) would discard). items.New(id) is used only when
// the slot's id is new or has changed. A slot the form clears (id <= 0)
// is left at the Worn zero value (empty), even if base had something there.
func mapToWornPreserving(base characters.Worn, submitted map[string]int) characters.Worn {
	baseByKey := make(map[string]*items.Item, 24)
	for _, s := range base.AllSlots() {
		baseByKey[s.Key] = s.Item
	}

	var w characters.Worn
	for _, s := range w.AllSlots() {
		id := submitted[s.Key]
		if id <= 0 {
			continue
		}
		if baseItem := baseByKey[s.Key]; baseItem != nil && baseItem.ItemId == id {
			*s.Item = *baseItem
			continue
		}
		*s.Item = items.New(id)
	}
	return w
}

// mergeCarriedItems rebuilds the carried-items list from the submitted ids,
// base-aware in the same spirit as mapToWornPreserving: a submitted id that
// matches an item the base template already carried reuses that EXACT
// instance (so its customization survives) rather than manufacturing a fresh
// spec-default one. Duplicate ids are handled correctly — each base instance
// is consumed at most once, so if the base carries two of the same item and
// the submission repeats that id twice, both original instances are reused
// (not the same one twice); a third repeat of that id falls through to
// items.New. Ids no longer present in the submission are simply dropped.
func mergeCarriedItems(base []items.Item, ids []int) []items.Item {
	pool := make(map[int][]items.Item, len(base))
	for _, it := range base {
		pool[it.ItemId] = append(pool[it.ItemId], it)
	}

	out := make([]items.Item, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if q := pool[id]; len(q) > 0 {
			out = append(out, q[0])
			pool[id] = q[1:]
			continue
		}
		out = append(out, items.New(id))
	}
	return out
}

// ---- LLM profile JSON round-trip ----
// The form shows the nested *llm.LLMProfile as an editable raw-JSON textarea
// (Task 6) rather than its own set of fields. llmProfileFromJSON returns an
// error on malformed input (rather than silently nil-ing the profile) so
// buildMobUpdate can surface it via buildErr instead of quietly discarding
// whatever the author typed.

func llmProfileToJSON(p *llm.LLMProfile) string {
	if p == nil {
		return ""
	}
	b, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return string(b)
}

func llmProfileFromJSON(s string) (*llm.LLMProfile, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var p llm.LLMProfile
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ---- senders ----

func sendMobList(uid int) {
	events.AddToQueue(GMCPOut{UserId: uid, Module: "Build.Mobs", Payload: buildMobList(realMobDeps())})
}
func sendMobDetail(uid int, d mobDetail) {
	events.AddToQueue(GMCPOut{UserId: uid, Module: "Build.Mob", Payload: d})
}
