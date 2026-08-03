package moderation

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

const (
	StatusOpen     = "open"
	StatusResolved = "resolved"
)

type Petition struct {
	Id         int       `yaml:"id"`
	Reporter   string    `yaml:"reporter"`
	Timestamp  time.Time `yaml:"timestamp"`
	RoomId     int       `yaml:"room_id"`
	Zone       string    `yaml:"zone"`
	Message    string    `yaml:"message"`
	Status     string    `yaml:"status"`
	ResolvedBy string    `yaml:"resolved_by,omitempty"`
	ResolvedAt time.Time `yaml:"resolved_at,omitempty"`
	Note       string    `yaml:"note,omitempty"`
}

var (
	petitions      []Petition
	nextPetitionId = 1
)

func petitionsPath() string { return filepath.Join(moderationDir(), "petitions.yaml") }

// Add records a new open petition, persists, and returns it.
func Add(reporter string, roomId int, zone, message string) (Petition, error) {
	mu.Lock()
	defer mu.Unlock()
	p := Petition{
		Id:        nextPetitionId,
		Reporter:  reporter,
		Timestamp: now(),
		RoomId:    roomId,
		Zone:      zone,
		Message:   message,
		Status:    StatusOpen,
	}
	petitions = append(petitions, p)
	nextPetitionId++
	if err := savePetitionsLocked(); err != nil {
		return p, err
	}
	return p, nil
}

func ListOpen() []Petition {
	mu.Lock()
	defer mu.Unlock()
	out := []Petition{}
	for _, p := range petitions {
		if p.Status == StatusOpen {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id }) // defensive: keep Id-ascending regardless of insert order
	return out
}

func ListAll() []Petition {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Petition, len(petitions))
	copy(out, petitions)
	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id }) // defensive: keep Id-ascending regardless of insert order
	return out
}

func Get(id int) (Petition, bool) {
	mu.Lock()
	defer mu.Unlock()
	for _, p := range petitions {
		if p.Id == id {
			return p, true
		}
	}
	return Petition{}, false
}

func Resolve(id int, by, note string) error {
	mu.Lock()
	defer mu.Unlock()
	for i := range petitions {
		if petitions[i].Id == id {
			if petitions[i].Status == StatusResolved {
				return fmt.Errorf("petition %d already resolved", id)
			}
			petitions[i].Status = StatusResolved
			petitions[i].ResolvedBy = by
			petitions[i].ResolvedAt = now()
			petitions[i].Note = note
			return savePetitionsLocked()
		}
	}
	return fmt.Errorf("no petition with id %d", id)
}

func savePetitionsLocked() error {
	if err := os.MkdirAll(moderationDir(), 0755); err != nil {
		mudlog.Error("moderation.savePetitions", "error", err.Error())
		return err
	}
	b, err := yaml.Marshal(petitions)
	if err != nil {
		mudlog.Error("moderation.savePetitions", "error", err.Error())
		return err
	}
	if err := os.WriteFile(petitionsPath(), b, 0644); err != nil {
		mudlog.Error("moderation.savePetitions", "error", err.Error())
		return err
	}
	return nil
}

// loadPetitions returns the number of petitions loaded.
func loadPetitions() int {
	mu.Lock()
	defer mu.Unlock()
	petitions = nil
	nextPetitionId = 1
	b, err := os.ReadFile(petitionsPath())
	if err != nil {
		return 0 // no file yet — fine
	}
	if err := yaml.Unmarshal(b, &petitions); err != nil {
		mudlog.Error("moderation.loadPetitions", "error", err.Error())
		petitions = nil
		return 0
	}
	for i, p := range petitions {
		if p.Id >= nextPetitionId {
			nextPetitionId = p.Id + 1
		}
		// Retroactive ANSI hygiene (2026-08-03): petitions filed before
		// write-side escaping shipped can hold raw <ansi> payloads.
		// Idempotent, so scrubbing every load is safe; the next save
		// persists the clean form.
		petitions[i].Message = util.EscapeAnsiTags(p.Message)
		petitions[i].Note = util.EscapeAnsiTags(p.Note)
	}
	return len(petitions)
}
