package gmcp

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/dialogue"
)

type fakeDialogueWorld struct {
	files   map[string]*dialogue.DialogueFile
	saved   []dialogue.DialogueFile
	created []string
	deleted []string
}

func dlgKey(mobId int, zone string) string { return fmt.Sprintf("%d:%s", mobId, zone) }

func newFakeDialogueWorld() *fakeDialogueWorld {
	return &fakeDialogueWorld{files: map[string]*dialogue.DialogueFile{
		dlgKey(9517, "Greenford"): {MobId: 9517, Zone: "Greenford", DefaultMood: "friendly"},
	}}
}

func (w *fakeDialogueWorld) deps() dialogueDeps {
	return dialogueDeps{
		load: func(mobId int, zone string) *dialogue.DialogueFile { return w.files[dlgKey(mobId, zone)] },
		save: func(df dialogue.DialogueFile) error {
			w.saved = append(w.saved, df)
			cp := df
			w.files[dlgKey(df.MobId, df.Zone)] = &cp
			return nil
		},
		create: func(mobId int, zone string) error {
			w.created = append(w.created, dlgKey(mobId, zone))
			w.files[dlgKey(mobId, zone)] = &dialogue.DialogueFile{MobId: mobId, Zone: zone}
			return nil
		},
		del: func(mobId int, zone string) error {
			w.deleted = append(w.deleted, dlgKey(mobId, zone))
			delete(w.files, dlgKey(mobId, zone))
			return nil
		},
		validators: dialogue.DialogueValidators{
			QuestExists:   func(string) bool { return true },
			QuestEndToken: func(string) (string, bool) { return "10-end", true },
			FlagDeclared:  func(string, string) bool { return true },
			ItemExists:    func(int) bool { return true },
		},
	}
}

func TestBuildDialogueUpdate_RefusesInvalidAndSavesNothing(t *testing.T) {
	w := newFakeDialogueWorld()
	df := dialogue.DialogueFile{MobId: 9517, Zone: "Greenford",
		Greetings: []dialogue.Greeting{{Text: "Hello; there."}}} // semicolon
	res := buildDialogueUpdate(w.deps(), df)
	if res.Ok {
		t.Error("invalid dialogue must be refused")
	}
	if len(w.saved) != 0 {
		t.Error("nothing may be saved when validation fails")
	}
}

func TestBuildDialogueUpdate_SavesAndSurfacesWarnings(t *testing.T) {
	w := newFakeDialogueWorld()
	df := dialogue.DialogueFile{MobId: 9517, Zone: "Greenford",
		Memory: dialogue.MemoryConfig{ExpiryPeriod: "2 real days"}} // warn, not error
	res := buildDialogueUpdate(w.deps(), df)
	if !res.Ok {
		t.Fatalf("warn-only file must save: %+v", res)
	}
	if len(res.Warnings) == 0 {
		t.Error("warnings must ride the successful result")
	}
	if len(w.saved) != 1 {
		t.Error("expected exactly one save")
	}
}

func TestBuildDialogueGetCreateDelete(t *testing.T) {
	w := newFakeDialogueWorld()
	if d, ok := buildDialogueGet(w.deps(), 9517, "Greenford"); !ok || d.File.DefaultMood != "friendly" {
		t.Errorf("get existing: ok=%v %+v", ok, d)
	}
	if _, ok := buildDialogueGet(w.deps(), 1, "Nowhere"); ok {
		t.Error("get missing file must report no-file (so the client offers Create)")
	}
	if res := buildDialogueCreate(w.deps(), 42, "Greenford"); !res.Ok || len(w.created) != 1 {
		t.Errorf("create: %+v", res)
	}
	if res := buildDialogueDelete(w.deps(), 9517, "Greenford"); !res.Ok || len(w.deleted) != 1 {
		t.Errorf("delete: %+v", res)
	}
}
