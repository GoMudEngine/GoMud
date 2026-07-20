// casing_sweep is a one-time tool that canonicalizes stored player-visible
// display names in the DOGMud data YAML files to smart Title Case.
//
// Only the VALUE of allowlisted display-name keys is rewritten. Comments,
// formatting, and all other YAML structure are preserved (no marshal
// round-trip). Run with -dry to preview without writing.
//
// Safety gate (verified before adding each entry to targets):
//
//	rooms  title:      display-only; rooms matched by ID.
//	buffs  name:       display-only; buffs matched by integer BuffId.
//	spells name:       FindSpellByName lowercases both sides (case-insensitive).
//	mobs   name:       findMobByName/MobIdByName use stringMatch (ToLower both).
//	items  name:       FindItemByName/NameMatch lower both sides (case-insensitive).
//	items  displayname: display-only; not used in any matcher.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/casing"
)

// targets maps a directory tree to the set of YAML keys whose VALUE is a
// player-visible display name.  Only keys listed here are rewritten.
// All matchers confirmed case-insensitive before inclusion (see safety gate
// in the file header).
var targets = map[string][]string{
	`_datafiles/world/dogmud/rooms`:  {"title"},
	`_datafiles/world/dogmud/buffs`:  {"name"},
	`_datafiles/world/dogmud/spells`: {"name"},
	`_datafiles/world/dogmud/mobs`:   {"name"}, // only one name: per mob file (under character:)
	`_datafiles/world/dogmud/items`:  {"name", "displayname"},
}

// keyLineRe builds a regex that matches a line with the given YAML key at any
// indentation, capturing prefix, value, and trailing whitespace separately.
func keyLineRe(key string) *regexp.Regexp {
	return regexp.MustCompile(`^(\s*` + regexp.QuoteMeta(key) + `:\s*)(.+?)(\s*)$`)
}

// canonicalizeValue applies casing.Title to the unquoted value.
// Returns the (possibly unchanged) raw string and whether it changed.
func canonicalizeValue(raw string) (string, bool) {
	v := strings.TrimSpace(raw)
	quoted := false
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		quoted = true
		v = v[1 : len(v)-1]
	}
	c := casing.Title(v)
	if c == v {
		return raw, false
	}
	if quoted {
		return `"` + c + `"`, true
	}
	return c, true
}

func main() {
	dry := flag.Bool("dry", false, "print changes without writing")
	flag.Parse()

	totalChanged := 0

	for dir, keys := range targets {
		res := make([]*regexp.Regexp, 0, len(keys))
		for _, k := range keys {
			res = append(res, keyLineRe(k))
		}
		dirChanged := 0
		_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".yaml") {
				return nil
			}
			n := processFile(p, res, *dry)
			dirChanged += n
			return nil
		})
		if dirChanged > 0 {
			fmt.Printf("[%s] %d line(s) changed\n", dir, dirChanged)
		}
		totalChanged += dirChanged
	}
	fmt.Printf("\nTotal lines changed: %d\n", totalChanged)
}

// processFile rewrites display-name lines in path. Returns the count of lines
// changed.
func processFile(path string, res []*regexp.Regexp, dry bool) int {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", path, err)
		return 0
	}
	lines := strings.Split(string(data), "\n")
	changed := 0
	for i, line := range lines {
		for _, re := range res {
			m := re.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			newVal, did := canonicalizeValue(m[2])
			if !did {
				continue
			}
			if dry {
				fmt.Printf("%s:%d\n  - %s\n  + %s%s\n", path, i+1, line, m[1], newVal)
			}
			lines[i] = m[1] + newVal
			changed++
			break
		}
	}
	if changed > 0 && !dry {
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", path, err)
		}
	}
	return changed
}
