package gmcp

import (
	"errors"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

type fakeBehaviorWorld struct {
	rows      []behaviortree.TreeFileRow
	defs      map[string]behaviortree.TreeDef // path -> def
	savedArch []string
	savedMob  []int
	savedRoom []int
	deleted   []string
	refs      []string
	saveErr   error
}

func probeTreeDef() behaviortree.TreeDef {
	return behaviortree.TreeDef{Tree: behaviortree.NodeDef{Type: "selector",
		Children: []behaviortree.NodeDef{{Type: "action", Do: "flee"}}}}
}

func (w *fakeBehaviorWorld) deps() behaviorDeps {
	return behaviorDeps{
		list:    func() []behaviortree.TreeFileRow { return w.rows },
		loadDef: func(path string) (behaviortree.TreeDef, error) { return w.defs[path], nil },
		saveArch: func(name string, d behaviortree.TreeDef) ([]string, error) {
			if w.saveErr != nil {
				return nil, w.saveErr
			}
			w.savedArch = append(w.savedArch, name)
			return []string{"root: empty selector — it always fails and does nothing"}, nil
		},
		saveMob: func(mobId int, zone, name string, d behaviortree.TreeDef) ([]string, error) {
			w.savedMob = append(w.savedMob, mobId)
			return nil, nil
		},
		saveRoom: func(roomId int, zone string, d behaviortree.TreeDef) ([]string, error) {
			w.savedRoom = append(w.savedRoom, roomId)
			return nil, nil
		},
		createArch: func(name string) error { w.savedArch = append(w.savedArch, name); return nil },
		delArch:    func(name string) error { w.deleted = append(w.deleted, "arch:"+name); return nil },
		delMob:     func(mobId int, zone, name string) error { w.deleted = append(w.deleted, "mob"); return nil },
		delRoom:    func(roomId int, zone string) error { w.deleted = append(w.deleted, "room"); return nil },
		archRefs:   func(name string) []string { return w.refs },
		archUsers: func(name string) []string {
			if name == "generic_fighter" {
				return []string{"74: Tunnel Shaman (Probe Zone)"}
			}
			return nil
		},
		hasComments: func(path string) bool { return strings.Contains(path, "commented") },
		mobSpec: func(id int) *mobs.Mob {
			if id != 74 {
				return nil
			}
			m := &mobs.Mob{MobId: 74, Zone: "Probe Zone", BehaviorArchetype: "generic_fighter"}
			m.Character.Name = "Tunnel Shaman"
			return m
		},
		roomTitle: func(id int) string { return "Probe Room" },
	}
}

func TestBuildBehaviorUpdate_ArchetypeSavesWithWarnings(t *testing.T) {
	w := &fakeBehaviorWorld{}
	res := buildBehaviorUpdate(w.deps(), behaviorUpdateReq{Kind: "archetype", Name: "generic_fighter", File: probeTreeDef()})
	if !res.Ok || len(w.savedArch) != 1 {
		t.Fatalf("archetype update should save: %+v", res)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("writer warnings must surface")
	}
}

func TestBuildBehaviorUpdate_RefusalPassesThrough(t *testing.T) {
	w := &fakeBehaviorWorld{saveErr: errors.New(`root: event "mob_death" is not fired by the engine`)}
	res := buildBehaviorUpdate(w.deps(), behaviorUpdateReq{Kind: "archetype", Name: "x", File: probeTreeDef()})
	if res.Ok || !strings.Contains(res.Error, "mob_death") {
		t.Fatalf("writer refusal must pass through verbatim: %+v", res)
	}
}

func TestBuildBehaviorUpdate_MobKindResolvesFromTemplate(t *testing.T) {
	w := &fakeBehaviorWorld{}
	res := buildBehaviorUpdate(w.deps(), behaviorUpdateReq{Kind: "mob", MobId: 74, File: probeTreeDef()})
	if !res.Ok || len(w.savedMob) != 1 {
		t.Fatalf("mob update should save via template zone/name: %+v", res)
	}
	if res := buildBehaviorUpdate(w.deps(), behaviorUpdateReq{Kind: "mob", MobId: 999, File: probeTreeDef()}); res.Ok {
		t.Fatal("unknown mob must refuse")
	}
}

func TestBuildBehaviorDelete_ArchetypeGuarded(t *testing.T) {
	w := &fakeBehaviorWorld{refs: []string{`mob 74 (Tunnel Shaman, Probe Zone): behavior_archetype`}}
	res := buildBehaviorDelete(w.deps(), behaviorGetReq{Kind: "archetype", Name: "generic_fighter"})
	if res.Ok || len(res.BehaviorRefs) == 0 || len(w.deleted) != 0 {
		t.Fatalf("referenced archetype must refuse with refs: %+v deleted=%v", res, w.deleted)
	}

	w.refs = nil
	res = buildBehaviorDelete(w.deps(), behaviorGetReq{Kind: "archetype", Name: "generic_fighter"})
	if !res.Ok || len(w.deleted) != 1 {
		t.Fatalf("unreferenced archetype should delete: %+v", res)
	}
}

func TestBuildBehaviorCreate_MobTreeSeedsFromArchetype(t *testing.T) {
	w := &fakeBehaviorWorld{defs: map[string]behaviortree.TreeDef{}}
	w.defs[behaviortree.GetArchetypePath("generic_fighter")] = probeTreeDef()
	res := buildBehaviorCreate(w.deps(), behaviorCreateReq{Kind: "mob", MobId: 74, FromArchetype: "generic_fighter"})
	if !res.Ok || len(w.savedMob) != 1 {
		t.Fatalf("mob-tree create should seed from the archetype and save: %+v", res)
	}
}

func TestBuildBehaviorList_ShapesAndUsedBy(t *testing.T) {
	w := &fakeBehaviorWorld{rows: []behaviortree.TreeFileRow{
		{Kind: "archetype", Name: "generic_fighter", Path: "commented/generic_fighter.yaml"},
		{Kind: "mob", MobId: 74, Zone: "probe_zone", Path: "x"},
		{Kind: "room", RoomId: 100, Zone: "probe_zone", Path: "y"},
	}}
	payload := buildBehaviorList(w.deps())
	if len(payload.Archetypes) != 1 || len(payload.MobTrees) != 1 || len(payload.RoomTrees) != 1 {
		t.Fatalf("list shape wrong: %+v", payload)
	}
	a := payload.Archetypes[0]
	if !a.HasHandComments || a.UsedBy == 0 {
		t.Fatalf("archetype row should carry hasHandComments + usedBy count: %+v", a)
	}
	if payload.MobTrees[0].MobName != "Tunnel Shaman" {
		t.Fatalf("mob tree rows should be decorated with the mob name: %+v", payload.MobTrees[0])
	}
}
