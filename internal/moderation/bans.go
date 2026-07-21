package moderation

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

type AccountBan struct {
	Username  string    `yaml:"username"`
	Reason    string    `yaml:"reason"`
	BannedBy  string    `yaml:"banned_by"`
	Timestamp time.Time `yaml:"timestamp"`
}

type IPBan struct {
	IP        string    `yaml:"ip"`
	Reason    string    `yaml:"reason"`
	BannedBy  string    `yaml:"banned_by"`
	Timestamp time.Time `yaml:"timestamp"`
}

// banFile is the on-disk shape of bans.yaml.
type banFile struct {
	Accounts []AccountBan `yaml:"accounts"`
	IPs      []IPBan      `yaml:"ips"`
}

var (
	accountBans = map[string]AccountBan{} // key: lowercased username
	ipBans      = map[string]IPBan{}      // key: exact host/ip
)

func bansPath() string { return filepath.Join(moderationDir(), "bans.yaml") }

// normAccountKey is the single source of truth for the account-ban map key, so
// the store, lookup, and reload paths can never drift (a drift would silently
// break bans across a restart).
func normAccountKey(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func BanAccount(username, reason, by string) error {
	mu.Lock()
	defer mu.Unlock()
	trimmed := strings.TrimSpace(username)
	accountBans[normAccountKey(trimmed)] = AccountBan{Username: trimmed, Reason: reason, BannedBy: by, Timestamp: now()}
	return saveBansLocked()
}

func Unban(username string) error {
	mu.Lock()
	defer mu.Unlock()
	delete(accountBans, normAccountKey(username))
	return saveBansLocked()
}

func IsAccountBanned(username string) (reason string, banned bool) {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := accountBans[normAccountKey(username)]; ok {
		return b.Reason, true
	}
	return "", false
}

func BanIP(ip, reason, by string) error {
	mu.Lock()
	defer mu.Unlock()
	ip = strings.TrimSpace(ip)
	ipBans[ip] = IPBan{IP: ip, Reason: reason, BannedBy: by, Timestamp: now()}
	return saveBansLocked()
}

func UnbanIP(ip string) error {
	mu.Lock()
	defer mu.Unlock()
	delete(ipBans, strings.TrimSpace(ip))
	return saveBansLocked()
}

func IsIPBanned(host string) (reason string, banned bool) {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := ipBans[strings.TrimSpace(host)]; ok {
		return b.Reason, true
	}
	return "", false
}

func saveBansLocked() error {
	if err := os.MkdirAll(moderationDir(), 0755); err != nil {
		mudlog.Error("moderation.saveBans", "error", err.Error())
		return err
	}
	bf := banFile{}
	for _, b := range accountBans {
		bf.Accounts = append(bf.Accounts, b)
	}
	for _, b := range ipBans {
		bf.IPs = append(bf.IPs, b)
	}
	out, err := yaml.Marshal(bf)
	if err != nil {
		mudlog.Error("moderation.saveBans", "error", err.Error())
		return err
	}
	if err := os.WriteFile(bansPath(), out, 0644); err != nil {
		mudlog.Error("moderation.saveBans", "error", err.Error())
		return err
	}
	return nil
}

// loadBans returns the number of account bans loaded.
func loadBans() int {
	mu.Lock()
	defer mu.Unlock()
	accountBans = map[string]AccountBan{}
	ipBans = map[string]IPBan{}
	b, err := os.ReadFile(bansPath())
	if err != nil {
		return 0
	}
	var bf banFile
	if err := yaml.Unmarshal(b, &bf); err != nil {
		mudlog.Error("moderation.loadBans", "error", err.Error())
		return 0
	}
	for _, a := range bf.Accounts {
		accountBans[normAccountKey(a.Username)] = a
	}
	for _, i := range bf.IPs {
		ipBans[i.IP] = i
	}
	return len(accountBans)
}
