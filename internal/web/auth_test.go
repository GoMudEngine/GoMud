package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

func TestDoBasicAuthRequiresAdminRole(t *testing.T) {
	const password = "correct-password"

	setupAuthTestUsers(t, password, map[string]string{
		"adminuser":   users.RoleAdmin,
		"builderuser": "builder",
		"normaluser":  users.RoleUser,
	})

	tests := []struct {
		name       string
		username   string
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "admin accepted",
			username:   "adminuser",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "builder rejected",
			username:   "builderuser",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "user rejected",
			username:   "normaluser",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAuthStateForTest()
			called := false
			handler := doBasicAuth(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
			req.SetBasicAuth(tt.username, password)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantCalled {
				t.Fatalf("handler called = %t, want %t", called, tt.wantCalled)
			}
		})
	}
}

// setupAuthTestUsers wires up a temporary DataFiles directory with an index
// and per-user YAML files, then redirects the configs DataFiles path to it
// for the duration of the test.
func setupAuthTestUsers(t *testing.T, password string, roles map[string]string) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	// ReloadConfig reads _datafiles/config.yaml from CWD; we must be at repo root.
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Chdir(%q): %v", repoRoot, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("Chdir cleanup: %v", err)
		}
	})

	// Restore real config after the test (registered before t.Setenv so it
	// runs after the env var is restored, i.e. without the override path set).
	t.Cleanup(func() {
		_ = configs.ReloadConfig()
		resetAuthStateForTest()
	})

	root := t.TempDir()
	dataDir := filepath.Join(root, "world")
	usersDir := filepath.Join(dataDir, "users")
	if err := os.MkdirAll(usersDir, 0700); err != nil {
		t.Fatalf("MkdirAll(%q): %v", usersDir, err)
	}

	// Use forward slashes so the YAML scalar is unambiguous on all platforms.
	dataDirSlash := filepath.ToSlash(dataDir)
	overridePath := filepath.Join(root, "config-overrides.yaml")
	overrideBytes := []byte("FilePaths:\n  DataFiles: " + dataDirSlash + "\n  CarefulSaveFiles: false\n")
	if err := os.WriteFile(overridePath, overrideBytes, 0600); err != nil {
		t.Fatalf("WriteFile(%q): %v", overridePath, err)
	}

	t.Setenv("CONFIG_PATH", filepath.ToSlash(overridePath))
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}

	// Build the binary user index now that DataFiles points to our temp dir.
	idx := users.NewUserIndex()
	if err := idx.Create(); err != nil {
		t.Fatalf("Create user index: %v", err)
	}

	userID := 1
	for username, role := range roles {
		if err := idx.AddUser(userID, username); err != nil {
			t.Fatalf("AddUser(%q): %v", username, err)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		if err != nil {
			t.Fatalf("GenerateFromPassword: %v", err)
		}

		u := users.NewUserRecord(userID, 0)
		u.Username = username
		u.Role = role
		u.Password = string(hash)

		data, err := yaml.Marshal(u)
		if err != nil {
			t.Fatalf("Marshal user %q: %v", username, err)
		}
		userPath := filepath.Join(usersDir, strconv.Itoa(userID)+".yaml")
		if err := os.WriteFile(userPath, data, 0600); err != nil {
			t.Fatalf("WriteFile(%q): %v", userPath, err)
		}

		userID++
	}
}

func resetAuthStateForTest() {
	authCache = map[string]time.Time{}
}
