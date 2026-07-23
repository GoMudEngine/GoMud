package gmcp

// Build.Item.* GMCP packages — the server side of the admin web item editor
// (admin web-building 2). Mirrors gmcp.Build.go (rooms): RoleAdmin-gated,
// deferred to MainWorker via GMCPBuildOp, core logic behind an itemDeps seam so
// it is unit-testable against a fake world.

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/statmods"
)

// ---- client -> server payloads ----

type itemGetReq struct {
	ItemId int `json:"itemId"`
}
type itemCreateReq struct {
	Type string `json:"type"`
}
type itemDeleteReq struct {
	ItemId int `json:"itemId"`
}

type itemSalvageRow struct {
	ItemTag  string `json:"itemTag"`
	Quantity int    `json:"quantity"`
}

// itemUpdateReq is both the Save payload and (embedded) the echoed detail.
type itemUpdateReq struct {
	ItemId           int            `json:"itemId"`
	Name             string         `json:"name"`
	DisplayName      string         `json:"displayName"`
	NameSimple       string         `json:"nameSimple"`
	Description      string         `json:"description"`
	Type             string         `json:"type"`
	Subtype          string         `json:"subtype"`
	Value            int            `json:"value"`
	Weight           float64        `json:"weight"`
	Uses             int            `json:"uses"`
	RarityTier       int            `json:"rarityTier"`
	VendorCategories []string       `json:"vendorCategories"`
	NotSalable       bool           `json:"notSalable"`
	NeverDrops       bool           `json:"neverDrops"`
	Restricted       bool           `json:"restricted"`
	Cursed           bool           `json:"cursed"`
	QuestToken       string         `json:"questToken"`
	StatMods         map[string]int `json:"statMods"`
	// weapon
	DamageMultiplier      float64 `json:"damageMultiplier"`
	SpellDamageMultiplier float64 `json:"spellDamageMultiplier"`
	Hands                 int     `json:"hands"`
	ParryRating           int     `json:"parryRating"`
	SpeedMultiplier       float64 `json:"speedMultiplier"`
	StaminaCost           int     `json:"staminaCost"`
	WaitRounds            int     `json:"waitRounds"`
	MinStrength           int     `json:"minStrength"`
	Reach                 float64 `json:"reach"`
	GrappleModifier       float64 `json:"grappleModifier"`
	Element               string  `json:"element"`
	AmmoTag               string  `json:"ammoTag"`
	// armor
	PhysicalMitigation   int     `json:"physicalMitigation"`
	MagicalMitigation    int     `json:"magicalMitigation"`
	ConvictionMitigation int     `json:"convictionMitigation"`
	BlockRating          int     `json:"blockRating"`
	EscapeModifier       float64 `json:"escapeModifier"`
	// consumable
	BuffIds               []int   `json:"buffIds"`
	Toxicity              int     `json:"toxicity"`
	FermentRounds         int     `json:"fermentRounds"`
	PeakRounds            int     `json:"peakRounds"`
	DecayRounds           int     `json:"decayRounds"`
	SpoilRounds           int     `json:"spoilRounds"`
	BottleAgingMultiplier float64 `json:"bottleAgingMultiplier"`
	IsBandolier           bool    `json:"isBandolier"`
	BandolierCapacity     int     `json:"bandolierCapacity"`
	// component
	IsComponent     bool             `json:"isComponent"`
	ComponentTag    string           `json:"componentTag"`
	WeightReduction float64          `json:"weightReduction"`
	BagCapacity     int              `json:"bagCapacity"`
	SalvageReturns  []itemSalvageRow `json:"salvageReturns"`
	// key
	KeyLockId string `json:"keyLockId"`
}

// ---- server -> client detail (Build.Item) ----
type itemDetail struct {
	itemUpdateReq          // same field set, echoed back
	Types         []string `json:"types"`
	Subtypes      []string `json:"subtypes"`
	Elements      []string `json:"elements"`
	Stats         []string `json:"stats"`
}

// itemListRow is one row of Build.Items.
type itemListRow struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Rarity  int    `json:"rarity"`
}

// ---- dependency seam ----
type itemRef struct {
	Kind string `json:"kind"` // mob | recipe | quest | container
	Id   string `json:"id"`   // e.g. "mob 9538 (a wolf)", "quest 10"
}

type itemDeps struct {
	load       func(id int) *items.ItemSpec
	save       func(s items.ItemSpec) error
	del        func(id int) error
	create     func(s items.ItemSpec) (int, error)
	references func(id int) []itemRef
}

func realItemDeps() itemDeps {
	return itemDeps{
		load:       items.GetItemSpec,
		save:       items.SaveItemSpec,
		del:        items.DeleteItemSpec,
		create:     items.CreateNewItemFile,
		references: scanItemReferences,
	}
}

// ---- core ----

func buildItemList(d itemDeps) []itemListRow {
	all := items.GetAllItemSpecs()
	rows := make([]itemListRow, 0, len(all))
	for _, s := range all {
		rows = append(rows, itemListRow{s.ItemId, s.Name, string(s.Type), string(s.Subtype), s.RarityTier})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Id < rows[j].Id })
	return rows
}

func specToReq(s *items.ItemSpec) itemUpdateReq {
	req := itemUpdateReq{
		ItemId: s.ItemId, Name: s.Name, DisplayName: s.DisplayName, NameSimple: s.NameSimple,
		Description: s.Description, Type: string(s.Type), Subtype: string(s.Subtype),
		Value: s.Value, Weight: s.Weight, Uses: s.Uses, RarityTier: s.RarityTier,
		VendorCategories: s.VendorCategories, NotSalable: s.NotSalable, NeverDrops: s.NeverDrops,
		Restricted: s.Restricted, Cursed: s.Cursed, QuestToken: s.QuestToken, StatMods: map[string]int(s.StatMods),
		DamageMultiplier: s.DamageMultiplier, SpellDamageMultiplier: s.SpellDamageMultiplier,
		Hands: s.Hands, ParryRating: s.ParryRating, SpeedMultiplier: s.SpeedMultiplier,
		StaminaCost: s.StaminaCost, WaitRounds: s.WaitRounds, MinStrength: s.MinStrength,
		Reach: s.Reach, GrappleModifier: s.GrappleModifier, Element: string(s.Element), AmmoTag: s.AmmoTag,
		PhysicalMitigation: s.PhysicalMitigation, MagicalMitigation: s.MagicalMitigation,
		ConvictionMitigation: s.ConvictionMitigation, BlockRating: s.BlockRating, EscapeModifier: s.EscapeModifier,
		BuffIds: s.BuffIds, Toxicity: s.Toxicity,
		FermentRounds: s.Aging.FermentRounds, PeakRounds: s.Aging.PeakRounds,
		DecayRounds: s.Aging.DecayRounds, SpoilRounds: s.Aging.SpoilRounds,
		BottleAgingMultiplier: s.BottleAgingMultiplier, IsBandolier: s.IsBandolier, BandolierCapacity: s.BandolierCapacity,
		IsComponent: s.IsComponent, ComponentTag: s.ComponentTag, WeightReduction: s.WeightReduction,
		BagCapacity: s.BagCapacity, KeyLockId: s.KeyLockId,
	}
	for _, sr := range s.SalvageReturns {
		req.SalvageReturns = append(req.SalvageReturns, itemSalvageRow{sr.ItemTag, sr.Quantity})
	}
	return req
}

func buildItemGet(d itemDeps, itemId int) (itemDetail, bool) {
	s := d.load(itemId)
	if s == nil {
		return itemDetail{}, false
	}
	return itemDetail{
		itemUpdateReq: specToReq(s),
		Types:         itemTypeIds(), Subtypes: itemSubtypeIds(), Elements: itemElementIds(), Stats: statModNames(),
	}, true
}

// reqToSpec starts from the loaded spec so fields the form does NOT cover
// (procs, reserves, worn-buffs, mutation drip, etc.) survive a Save untouched.
func reqToSpec(base *items.ItemSpec, req itemUpdateReq) items.ItemSpec {
	s := *base
	s.Name, s.DisplayName, s.NameSimple, s.Description = req.Name, req.DisplayName, req.NameSimple, req.Description
	s.Type, s.Subtype = items.ItemType(req.Type), items.ItemSubType(req.Subtype)
	s.Value, s.Weight, s.Uses, s.RarityTier = req.Value, req.Weight, req.Uses, req.RarityTier
	s.VendorCategories, s.QuestToken, s.StatMods = req.VendorCategories, req.QuestToken, statmods.StatMods(req.StatMods)
	s.NotSalable, s.NeverDrops, s.Restricted, s.Cursed = req.NotSalable, req.NeverDrops, req.Restricted, req.Cursed
	s.DamageMultiplier, s.SpellDamageMultiplier, s.Hands = req.DamageMultiplier, req.SpellDamageMultiplier, req.Hands
	s.ParryRating, s.SpeedMultiplier, s.StaminaCost, s.WaitRounds = req.ParryRating, req.SpeedMultiplier, req.StaminaCost, req.WaitRounds
	s.MinStrength, s.Reach, s.GrappleModifier = req.MinStrength, req.Reach, req.GrappleModifier
	s.Element, s.AmmoTag = items.Element(req.Element), req.AmmoTag
	s.PhysicalMitigation, s.MagicalMitigation, s.ConvictionMitigation = req.PhysicalMitigation, req.MagicalMitigation, req.ConvictionMitigation
	s.BlockRating, s.EscapeModifier = req.BlockRating, req.EscapeModifier
	s.BuffIds, s.Toxicity = req.BuffIds, req.Toxicity
	s.Aging = items.AgingThresholds{FermentRounds: req.FermentRounds, PeakRounds: req.PeakRounds, DecayRounds: req.DecayRounds, SpoilRounds: req.SpoilRounds}
	s.BottleAgingMultiplier, s.IsBandolier, s.BandolierCapacity = req.BottleAgingMultiplier, req.IsBandolier, req.BandolierCapacity
	s.IsComponent, s.ComponentTag, s.WeightReduction, s.BagCapacity = req.IsComponent, req.ComponentTag, req.WeightReduction, req.BagCapacity
	s.KeyLockId = req.KeyLockId
	s.SalvageReturns = nil
	for _, r := range req.SalvageReturns {
		if r.ItemTag != "" {
			s.SalvageReturns = append(s.SalvageReturns, items.SalvageReturn{ItemTag: r.ItemTag, Quantity: r.Quantity})
		}
	}
	return s
}

func buildItemUpdate(d itemDeps, req itemUpdateReq) BuildResult {
	base := d.load(req.ItemId)
	if base == nil {
		return buildErr("item %d not found", req.ItemId)
	}
	if req.Name == "" || req.Type == "" {
		return buildErr("name and type are required")
	}
	if err := d.save(reqToSpec(base, req)); err != nil {
		return buildErr("could not save item %d: %s", req.ItemId, err.Error())
	}
	return BuildResult{Ok: true, ItemId: req.ItemId}
}

func buildItemDelete(d itemDeps, itemId int) BuildResult {
	if d.load(itemId) == nil {
		return buildErr("item %d not found", itemId)
	}
	if refs := d.references(itemId); len(refs) > 0 {
		return BuildResult{Error: "item is still referenced — remove these first", Refs: refs}
	}
	if err := d.del(itemId); err != nil {
		return buildErr("could not delete item %d: %s", itemId, err.Error())
	}
	return BuildResult{Ok: true, ItemId: itemId}
}

func buildItemCreate(d itemDeps, itemType string) BuildResult {
	if itemType == "" {
		return buildErr("choose an item type")
	}
	// Name must be canonical Title casing or the item loader panics on the next
	// boot (CanonicalizeItemNames in the items package enforces this on save too).
	seed := items.ItemSpec{Type: items.ItemType(itemType), Name: "New Item", Description: "An unfinished item.", Hands: 1}
	id, err := d.create(seed)
	if err != nil {
		return buildErr("%s", err.Error())
	}
	return BuildResult{Ok: true, ItemId: id}
}

// ---- enum providers ----

func itemTypeIds() []string {
	out := []string{}
	for _, t := range items.ItemTypes() {
		out = append(out, t.Type)
	}
	return dedupeSorted(out)
}
func itemSubtypeIds() []string {
	out := []string{}
	for _, t := range items.ItemSubtypes() {
		out = append(out, t.Type)
	}
	return dedupeSorted(out)
}
func itemElementIds() []string {
	return []string{"", "fire", "water", "ice", "electricity", "acid", "life", "death"}
}
func statModNames() []string {
	return []string{
		"strength", "dexterity", "perception", "vitality", "willpower", "charisma",
		"weapon-combat", "unarmed-combat", "ranged-combat", "spellcasting", "rhetoric", "manifestation", "skullduggery", "salvage",
	}
}
func dedupeSorted(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// ---- senders ----

func sendItemList(uid int) {
	events.AddToQueue(GMCPOut{UserId: uid, Module: "Build.Items", Payload: buildItemList(realItemDeps())})
}
func sendItemDetail(uid int, d itemDetail) {
	events.AddToQueue(GMCPOut{UserId: uid, Module: "Build.Item", Payload: d})
}
