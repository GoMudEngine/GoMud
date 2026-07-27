package gmcp

// Build.Behavior.* GMCP packages — the server side of the admin web
// behavior-tree editor (admin web-building 5d). Three tree families ride one
// verb set, kind-routed: "archetype" (name), "mob" (mobId — zone and name
// resolve from the LIVE template, never from the client), "room" (roomId +
// zone). Saves hot-reload the btree engine (the writer owns that contract);
// there is no reindex step here.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

type behaviorDeps struct {
	list        func() []behaviortree.TreeFileRow
	loadDef     func(path string) (behaviortree.TreeDef, error)
	saveArch    func(name string, d behaviortree.TreeDef) ([]string, error)
	saveMob     func(mobId int, zone, name string, d behaviortree.TreeDef) ([]string, error)
	saveRoom    func(roomId int, zone string, d behaviortree.TreeDef) ([]string, error)
	createArch  func(name string) error
	delArch     func(name string) error
	delMob      func(mobId int, zone, name string) error
	delRoom     func(roomId int, zone string) error
	archRefs    func(name string) []string
	archUsers   func(name string) []string
	hasComments func(path string) bool
	mobSpec     func(id int) *mobs.Mob
	roomTitle   func(id int) string
}

func realBehaviorDeps() behaviorDeps {
	return behaviorDeps{
		list:       behaviortree.ListTreeFiles,
		loadDef:    behaviortree.LoadTreeDef,
		saveArch:   behaviortree.SaveArchetype,
		saveMob:    behaviortree.SaveMobTree,
		saveRoom:   behaviortree.SaveRoomTree,
		createArch: behaviortree.CreateArchetype,
		delArch:    behaviortree.DeleteArchetype,
		delMob:     behaviortree.DeleteMobTree,
		delRoom:    behaviortree.DeleteRoomTree,
		archRefs:   behaviortree.ArchetypeReferences,
		archUsers: func(name string) []string {
			out := []string{}
			for _, m := range mobs.AllMobTemplates() {
				if m.BehaviorArchetype == name {
					out = append(out, fmt.Sprintf("%d: %s (%s)", int(m.MobId), m.Character.Name, m.Zone))
				}
			}
			sort.Strings(out)
			return out
		},
		hasComments: behaviortree.RawFileHasHandComments,
		mobSpec:     func(id int) *mobs.Mob { return mobs.GetMobSpec(mobs.MobId(id)) },
		roomTitle: func(id int) string {
			if r := rooms.LoadRoomTemplate(id); r != nil {
				return r.Title
			}
			return ""
		},
	}
}

// ---- payloads ----

type behaviorGetReq struct {
	Kind   string `json:"kind"` // archetype | mob | room
	Name   string `json:"name,omitempty"`
	MobId  int    `json:"mobId,omitempty"`
	RoomId int    `json:"roomId,omitempty"`
	Zone   string `json:"zone,omitempty"` // room kind only
}

type behaviorCreateReq struct {
	Kind          string `json:"kind"`
	Name          string `json:"name,omitempty"`          // archetype kind
	MobId         int    `json:"mobId,omitempty"`         // mob kind
	FromArchetype string `json:"fromArchetype,omitempty"` // mob kind: seed source
	RoomId        int    `json:"roomId,omitempty"`        // room kind
	Zone          string `json:"zone,omitempty"`          // room kind
}

type behaviorUpdateReq struct {
	Kind   string               `json:"kind"`
	Name   string               `json:"name,omitempty"`
	MobId  int                  `json:"mobId,omitempty"`
	RoomId int                  `json:"roomId,omitempty"`
	Zone   string               `json:"zone,omitempty"`
	File   behaviortree.TreeDef `json:"file"`
}

type behaviorArchRow struct {
	Name            string `json:"name"`
	UsedBy          int    `json:"usedBy"`
	HasHandComments bool   `json:"hasHandComments"`
}

type behaviorMobRow struct {
	MobId   int    `json:"mobId"`
	MobName string `json:"mobName"`
	Zone    string `json:"zone"`
}

type behaviorRoomRow struct {
	RoomId int    `json:"roomId"`
	Zone   string `json:"zone"`
	Title  string `json:"title"`
}

type behaviorListPayload struct {
	Archetypes []behaviorArchRow `json:"archetypes"`
	MobTrees   []behaviorMobRow  `json:"mobTrees"`
	RoomTrees  []behaviorRoomRow `json:"roomTrees"`
}

type behaviorEnums struct {
	NodeTypes     []string     `json:"nodeTypes"`
	DecoratorMods []vocabEntry `json:"decoratorMods"` // description names the param key
	Conditions    []string     `json:"conditions"`
	Actions       []string     `json:"actions"`
	Events        []string     `json:"events"`
	Archetypes    []string     `json:"archetypes"`
}

type behaviorDetail struct {
	Kind            string               `json:"kind"`
	Name            string               `json:"name,omitempty"`
	MobId           int                  `json:"mobId,omitempty"`
	MobName         string               `json:"mobName,omitempty"`
	RoomId          int                  `json:"roomId,omitempty"`
	Zone            string               `json:"zone,omitempty"`
	File            behaviortree.TreeDef `json:"file"`
	Found           bool                 `json:"found"`
	HasHandComments bool                 `json:"hasHandComments"`
	UsedBy          []string             `json:"usedBy,omitempty"` // archetype kind
	Enums           behaviorEnums        `json:"enums"`
}

// ---- handlers ----

func buildBehaviorList(d behaviorDeps) behaviorListPayload {
	p := behaviorListPayload{Archetypes: []behaviorArchRow{}, MobTrees: []behaviorMobRow{}, RoomTrees: []behaviorRoomRow{}}
	for _, r := range d.list() {
		switch r.Kind {
		case "archetype":
			p.Archetypes = append(p.Archetypes, behaviorArchRow{
				Name: r.Name, UsedBy: len(d.archUsers(r.Name)), HasHandComments: d.hasComments(r.Path)})
		case "mob":
			name := ""
			if m := d.mobSpec(r.MobId); m != nil {
				name = m.Character.Name
			}
			p.MobTrees = append(p.MobTrees, behaviorMobRow{MobId: r.MobId, MobName: name, Zone: r.Zone})
		case "room":
			p.RoomTrees = append(p.RoomTrees, behaviorRoomRow{RoomId: r.RoomId, Zone: r.Zone, Title: d.roomTitle(r.RoomId)})
		}
	}
	sort.Slice(p.Archetypes, func(i, j int) bool { return p.Archetypes[i].Name < p.Archetypes[j].Name })
	sort.Slice(p.MobTrees, func(i, j int) bool { return p.MobTrees[i].MobId < p.MobTrees[j].MobId })
	sort.Slice(p.RoomTrees, func(i, j int) bool { return p.RoomTrees[i].RoomId < p.RoomTrees[j].RoomId })
	return p
}

// behaviorPathFor resolves the on-disk path for a get/update request. Mob
// kind resolves zone+name from the LIVE template — the client never names
// paths.
func behaviorPathFor(d behaviorDeps, kind string, name string, mobId, roomId int, zone string) (string, behaviorDetail, error) {
	det := behaviorDetail{Kind: kind}
	switch kind {
	case "archetype":
		if name == "" {
			return "", det, fmt.Errorf("archetype kind needs a name")
		}
		det.Name = name
		return behaviortree.GetArchetypePath(name), det, nil
	case "mob":
		m := d.mobSpec(mobId)
		if m == nil {
			return "", det, fmt.Errorf("mob %d not found", mobId)
		}
		det.MobId = mobId
		det.MobName = m.Character.Name
		det.Zone = m.Zone
		return behaviortree.GetBehaviorPath(mobId, m.Zone, m.Character.Name), det, nil
	case "room":
		if roomId == 0 {
			return "", det, fmt.Errorf("room kind needs a roomId")
		}
		det.RoomId = roomId
		det.Zone = zone
		return behaviortree.GetRoomBehaviorPath(roomId, zone), det, nil
	}
	return "", det, fmt.Errorf("unknown behavior kind %q", kind)
}

func buildBehaviorGet(d behaviorDeps, req behaviorGetReq) (behaviorDetail, bool) {
	path, det, err := behaviorPathFor(d, req.Kind, req.Name, req.MobId, req.RoomId, req.Zone)
	det.Enums = collectBehaviorEnums(d)
	if err != nil {
		return det, false
	}
	def, lerr := d.loadDef(path)
	if lerr != nil {
		return det, false
	}
	det.File = def
	det.Found = true
	det.HasHandComments = d.hasComments(path)
	if req.Kind == "archetype" {
		det.UsedBy = d.archUsers(req.Name)
	}
	return det, true
}

func buildBehaviorUpdate(d behaviorDeps, req behaviorUpdateReq) BuildResult {
	var warns []string
	var err error
	switch req.Kind {
	case "archetype":
		if req.Name == "" {
			return buildErr("archetype kind needs a name")
		}
		warns, err = d.saveArch(req.Name, req.File)
	case "mob":
		m := d.mobSpec(req.MobId)
		if m == nil {
			return buildErr("mob %d not found", req.MobId)
		}
		warns, err = d.saveMob(req.MobId, m.Zone, m.Character.Name, req.File)
	case "room":
		if req.RoomId == 0 {
			return buildErr("room kind needs a roomId")
		}
		warns, err = d.saveRoom(req.RoomId, req.Zone, req.File)
	default:
		return buildErr("unknown behavior kind %q", req.Kind)
	}
	if err != nil {
		return buildErr("behavior refused: %s", err.Error())
	}
	return BuildResult{Ok: true, Message: "behavior saved — live immediately", Warnings: warns}
}

func buildBehaviorCreate(d behaviorDeps, req behaviorCreateReq) BuildResult {
	switch req.Kind {
	case "archetype":
		if req.Name == "" {
			return buildErr("archetype kind needs a name")
		}
		if err := d.createArch(req.Name); err != nil {
			return buildErr("%s", err.Error())
		}
		return BuildResult{Ok: true, Message: fmt.Sprintf("archetype %q created", req.Name)}
	case "mob":
		m := d.mobSpec(req.MobId)
		if m == nil {
			return buildErr("mob %d not found", req.MobId)
		}
		seed := behaviortree.TreeDef{Tree: behaviortree.NodeDef{Type: "selector"}}
		if req.FromArchetype != "" {
			def, err := d.loadDef(behaviortree.GetArchetypePath(req.FromArchetype))
			if err != nil {
				return buildErr("seed archetype %q unreadable: %s", req.FromArchetype, err.Error())
			}
			seed = def
			seed.Notes = strings.TrimSpace(fmt.Sprintf("Specialized from archetype %q. %s", req.FromArchetype, def.Notes))
		}
		if _, err := d.saveMob(req.MobId, m.Zone, m.Character.Name, seed); err != nil {
			return buildErr("%s", err.Error())
		}
		return BuildResult{Ok: true, MobId: req.MobId, Message: fmt.Sprintf("per-mob tree created for %s — it now OVERRIDES the archetype", m.Character.Name)}
	case "room":
		if req.RoomId == 0 {
			return buildErr("room kind needs a roomId")
		}
		seed := behaviortree.TreeDef{
			Notes: "Describe what this room's tree does here.",
			Tree: behaviortree.NodeDef{Type: "selector", Children: []behaviortree.NodeDef{
				{Type: "action", Event: "room_idle", Do: "room_message", Note: "example node — replace",
					Params: map[string]any{"text": "The room stirs."}},
			}},
		}
		if _, err := d.saveRoom(req.RoomId, req.Zone, seed); err != nil {
			return buildErr("%s", err.Error())
		}
		return BuildResult{Ok: true, RoomId: req.RoomId, Message: fmt.Sprintf("room tree created for room %d", req.RoomId)}
	}
	return buildErr("unknown behavior kind %q", req.Kind)
}

func buildBehaviorDelete(d behaviorDeps, req behaviorGetReq) BuildResult {
	switch req.Kind {
	case "archetype":
		if refs := d.archRefs(req.Name); len(refs) > 0 {
			return BuildResult{Ok: false, BehaviorRefs: refs,
				Error: fmt.Sprintf("archetype %q is still referenced:\n%s", req.Name, strings.Join(refs, "\n"))}
		}
		if err := d.delArch(req.Name); err != nil {
			return buildErr("%s", err.Error())
		}
		return BuildResult{Ok: true, Message: fmt.Sprintf("archetype %q deleted", req.Name)}
	case "mob":
		m := d.mobSpec(req.MobId)
		if m == nil {
			return buildErr("mob %d not found", req.MobId)
		}
		if err := d.delMob(req.MobId, m.Zone, m.Character.Name); err != nil {
			return buildErr("%s", err.Error())
		}
		return BuildResult{Ok: true, MobId: req.MobId, Message: fmt.Sprintf("per-mob tree deleted — %s falls back to its archetype", m.Character.Name)}
	case "room":
		if err := d.delRoom(req.RoomId, req.Zone); err != nil {
			return buildErr("%s", err.Error())
		}
		return BuildResult{Ok: true, RoomId: req.RoomId, Message: fmt.Sprintf("room tree deleted for room %d", req.RoomId)}
	}
	return buildErr("unknown behavior kind %q", req.Kind)
}

// ---- enums ----

func collectBehaviorEnums(d behaviorDeps) behaviorEnums {
	e := behaviorEnums{
		NodeTypes: []string{"selector", "sequence", "condition", "action", "decorator"},
		DecoratorMods: []vocabEntry{
			{"cooldown", "skip the child while cooling down (param: rounds)"},
			{"repeat", "run the child N times (param: times)"},
			{"invert", "flip the child's success/failure"},
			{"random", "run the child with a % chance (param: percent)"},
			{"delay", "wait before running the child (param: rounds)"},
		},
		Conditions: behaviortree.ConditionNames(),
		Actions:    behaviortree.ActionNames(),
		Events:     behaviortree.EventNames(),
	}
	for _, r := range d.list() {
		if r.Kind == "archetype" {
			e.Archetypes = append(e.Archetypes, r.Name)
		}
	}
	sort.Strings(e.Archetypes)
	return e
}
