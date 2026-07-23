package items

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/casing"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
)

// CanonicalizeItemNames normalizes the player-visible name fields to the Title
// casing the item loader demands. LoadDataFiles() hard-panics via
// casing.AssertCanonical on any non-canonical Name/DisplayName, so a single
// non-canonical save (or an unedited "new item" seed) bricks the NEXT boot.
// Every write path must run this first. Color-tagged display names are left
// alone: Title() would split on the tag's whitespace and mangle it, and such
// names are already accepted by the loader.
func CanonicalizeItemNames(spec *ItemSpec) {
	spec.Name = casing.Title(spec.Name)
	if spec.DisplayName != "" && !strings.Contains(spec.DisplayName, "<") {
		spec.DisplayName = casing.Title(spec.DisplayName)
	}
}

// itemsBasePath is the root SaveFlatFile joins with ItemSpec.Filepath(). It is a
// package var so tests can point it at a temp dir without touching the config
// override file. Mirrors CreateNewItemFile's `DataFiles + "/items"` base.
var itemsBasePath = func() string {
	return configs.GetFilePathsConfig().DataFiles.String() + `/items`
}

// SaveItemSpec validates and writes an existing/edited item template, updating
// the in-memory cache. Item filenames embed the name (<id>-<name>.yaml) and
// armor lives in per-type subfolders, so a rename or retype changes the file
// path — the OLD file is removed first so a stale duplicate can't linger (and
// boot as a second copy).
func SaveItemSpec(spec ItemSpec) error {
	CanonicalizeItemNames(&spec)
	if err := spec.Validate(); err != nil {
		return err
	}
	base := itemsBasePath()
	newRel := spec.Filepath()

	if old, ok := items[spec.ItemId]; ok {
		if oldRel := old.Filepath(); oldRel != newRel {
			if err := os.Remove(filepath.Join(base, filepath.FromSlash(oldRel))); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}
	if err := fileloader.SaveFlatFile[*ItemSpec](base, &spec, saveModes...); err != nil {
		return err
	}

	cp := spec
	items[spec.ItemId] = &cp
	return nil
}

// DeleteItemSpec removes an item template's file and cache entry. Callers must
// first confirm the item is unreferenced (the web builder's reference scan).
func DeleteItemSpec(itemId int) error {
	spec, ok := items[itemId]
	if !ok {
		return fmt.Errorf(`item %d not found`, itemId)
	}
	if err := os.Remove(filepath.Join(itemsBasePath(), filepath.FromSlash(spec.Filepath()))); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(items, itemId)
	return nil
}
