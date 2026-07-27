package behaviortree

// The behavior-tree writer (admin web-building 5d). Validation rides the
// loader's own compiler — the writer marshals, then compiles its own output,
// so nothing the loader would refuse at boot can reach disk — plus the
// event-vocabulary check the runtime never had. Saves go live immediately:
// the engine's Load* calls replace the positive cache and clear the negative
// (this is the hot-reload the engine.go negative-cache design note warned
// about, done deliberately); deletes evict AND set the negative, which is
// exactly the fallback semantics.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"gopkg.in/yaml.v2"
)

// validateTreeDef marshals the definition and compiles the result (the
// loader is the structural validator), then enforces the event vocabulary.
// Returns the marshaled bytes on success so save paths write exactly what
// was validated.
func validateTreeDef(d TreeDef) ([]byte, []string, error) {
	out, err := yaml.Marshal(d)
	if err != nil {
		return nil, nil, err
	}
	if _, err := LoadTreeFromBytes(out); err != nil {
		return nil, nil, err
	}

	var warns []string
	var walk func(n NodeDef, path string) error
	walk = func(n NodeDef, path string) error {
		if n.Event != "" && !KnownBehaviorEvents[n.Event] {
			return fmt.Errorf("%s: event %q is not fired by the engine — this node would NEVER run (valid: %s)",
				path, n.Event, strings.Join(EventNames(), ", "))
		}
		if (n.Type == "selector" || n.Type == "sequence") && len(n.Children) == 0 {
			warns = append(warns, fmt.Sprintf("%s: empty %s — it always fails and does nothing", path, n.Type))
		}
		for i, ch := range n.Children {
			if err := walk(ch, fmt.Sprintf("%s.%s[%d]", path, n.Type, i)); err != nil {
				return err
			}
		}
		if n.Child != nil {
			if err := walk(*n.Child, path+".child"); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(d.Tree, "root"); err != nil {
		return nil, nil, err
	}
	return out, warns, nil
}

func writeTreeFile(path string, out []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// SaveArchetype validates, writes, and hot-reloads an archetype.
func SaveArchetype(name string, d TreeDef) ([]string, error) {
	out, warns, err := validateTreeDef(d)
	if err != nil {
		return nil, err
	}
	path := GetArchetypePath(name)
	if err := writeTreeFile(path, out); err != nil {
		return nil, err
	}
	if err := GetEngine().LoadArchetype(name, path); err != nil {
		return nil, fmt.Errorf("written but reload failed (cache may be stale): %w", err)
	}
	return warns, nil
}

// SaveMobTree validates, writes, and hot-reloads a per-mob tree. The path
// embeds the mob's live name — callers pass it from the template.
func SaveMobTree(mobId int, zone string, mobName string, d TreeDef) ([]string, error) {
	out, warns, err := validateTreeDef(d)
	if err != nil {
		return nil, err
	}
	path := GetBehaviorPath(mobId, zone, mobName)
	if err := writeTreeFile(path, out); err != nil {
		return nil, err
	}
	if err := GetEngine().LoadTree(mobId, path); err != nil {
		return nil, fmt.Errorf("written but reload failed (cache may be stale): %w", err)
	}
	return warns, nil
}

// SaveRoomTree validates, writes, and hot-reloads a room tree.
func SaveRoomTree(roomId int, zone string, d TreeDef) ([]string, error) {
	out, warns, err := validateTreeDef(d)
	if err != nil {
		return nil, err
	}
	path := GetRoomBehaviorPath(roomId, zone)
	if err := writeTreeFile(path, out); err != nil {
		return nil, err
	}
	if err := GetEngine().LoadRoomTree(roomId, path); err != nil {
		return nil, fmt.Errorf("written but reload failed (cache may be stale): %w", err)
	}
	return warns, nil
}

// CreateArchetype refuses an existing archetype and writes a minimal
// compiling skeleton.
func CreateArchetype(name string) error {
	path := GetArchetypePath(name)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("archetype %q already exists", name)
	}
	skeleton := TreeDef{
		Notes: "Describe this archetype's fighting style / role here.",
		Tree: NodeDef{Type: "selector", Children: []NodeDef{
			{Type: "action", Event: "mob_hurt", Do: "flee",
				Note: "example node — replace: panic-flee whenever hurt"},
		}},
	}
	_, err := SaveArchetype(name, skeleton)
	return err
}

// DeleteArchetype removes the file and evicts (positive + goal maps +
// negative set). The reference guard is the caller's job.
func DeleteArchetype(name string) error {
	if err := os.Remove(GetArchetypePath(name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	GetEngine().EvictArchetype(name)
	return nil
}

// DeleteMobTree removes a per-mob tree; the mob falls back to its archetype
// (the negative cache is CORRECT after delete — that IS the fallback).
func DeleteMobTree(mobId int, zone string, mobName string) error {
	if err := os.Remove(GetBehaviorPath(mobId, zone, mobName)); err != nil && !os.IsNotExist(err) {
		return err
	}
	GetEngine().EvictTree(mobId)
	return nil
}

// DeleteRoomTree removes a room tree.
func DeleteRoomTree(roomId int, zone string) error {
	if err := os.Remove(GetRoomBehaviorPath(roomId, zone)); err != nil && !os.IsNotExist(err) {
		return err
	}
	GetEngine().EvictRoomTree(roomId)
	return nil
}

// MoveMobBehaviorFile relocates a per-mob tree when its mob is renamed or
// re-zoned (the path embeds both), then reloads the cache from the new
// path. Registered on mobs.OnMobFileRename at init. Silently a no-op when
// the mob has no behavior file — which is most mobs.
func MoveMobBehaviorFile(mobId int, oldZone, oldName, newZone, newName string) {
	oldPath := GetBehaviorPath(mobId, oldZone, oldName)
	if _, err := os.Stat(oldPath); err != nil {
		return
	}
	newPath := GetBehaviorPath(mobId, newZone, newName)
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return
	}
	_ = GetEngine().LoadTree(mobId, newPath)
}

// RawFileHasHandComments reports whether the on-disk file carries full-line
// `#` comments — the 5d editor's marshal drops them, so the panel warns
// before the first save. Marshal output never contains them, so this is
// also "has this file ever been editor-saved".
func RawFileHasHandComments(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			return true
		}
	}
	return false
}

// behaviorsRoot is the top-level behaviors/ directory.
func behaviorsRoot() string {
	return filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "behaviors")
}

// LoadTreeDef parses a tree file WITHOUT compiling it — the editor's Get
// path (the compile happens on save; a broken file should still open so it
// can be fixed).
func LoadTreeDef(path string) (TreeDef, error) {
	var d TreeDef
	data, err := os.ReadFile(path)
	if err != nil {
		return d, err
	}
	if err := yaml.Unmarshal(data, &d); err != nil {
		return d, fmt.Errorf("parse error: %w", err)
	}
	return d, nil
}

// TreeFileRow describes one on-disk behavior file for the editor's list.
type TreeFileRow struct {
	Kind   string // archetype | mob | room
	Name   string // archetype name (archetype kind)
	MobId  int    // mob kind
	RoomId int    // room kind
	Zone   string // mob/room kinds
	Path   string
}

// ListTreeFiles walks the behaviors tree and returns every archetype,
// per-mob, and room tree file.
func ListTreeFiles() []TreeFileRow {
	rows := []TreeFileRow{}
	root := behaviorsRoot()

	// archetypes/<name>.yaml
	if entries, err := os.ReadDir(filepath.Join(root, "archetypes")); err == nil {
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".yaml")
			rows = append(rows, TreeFileRow{Kind: "archetype", Name: name,
				Path: filepath.Join(root, "archetypes", e.Name())})
		}
	}

	zoneDirs, err := os.ReadDir(root)
	if err != nil {
		return rows
	}
	for _, zd := range zoneDirs {
		if !zd.IsDir() || zd.Name() == "archetypes" || zd.Name() == "rooms" {
			continue
		}
		// <zone>/<mobId>-<name>.yaml
		if files, err := os.ReadDir(filepath.Join(root, zd.Name())); err == nil {
			for _, f := range files {
				if f.IsDir() || filepath.Ext(f.Name()) != ".yaml" {
					continue
				}
				var mobId int
				if _, err := fmt.Sscanf(f.Name(), "%d-", &mobId); err != nil || mobId <= 0 {
					continue
				}
				rows = append(rows, TreeFileRow{Kind: "mob", MobId: mobId, Zone: zd.Name(),
					Path: filepath.Join(root, zd.Name(), f.Name())})
			}
		}
	}

	// rooms/<zone>/<roomId>.yaml
	if roomZones, err := os.ReadDir(filepath.Join(root, "rooms")); err == nil {
		for _, zd := range roomZones {
			if !zd.IsDir() {
				continue
			}
			if files, err := os.ReadDir(filepath.Join(root, "rooms", zd.Name())); err == nil {
				for _, f := range files {
					if f.IsDir() || filepath.Ext(f.Name()) != ".yaml" {
						continue
					}
					var roomId int
					if _, err := fmt.Sscanf(f.Name(), "%d.yaml", &roomId); err != nil || roomId <= 0 {
						continue
					}
					rows = append(rows, TreeFileRow{Kind: "room", RoomId: roomId, Zone: zd.Name(),
						Path: filepath.Join(root, "rooms", zd.Name(), f.Name())})
				}
			}
		}
	}
	return rows
}
