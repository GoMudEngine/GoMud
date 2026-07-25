package rooms

// ZoneFolderCollision reports the first zone in existing whose sanitized
// folder name matches newName's, or "" when the folder is free.
//
// ZoneNameSanitize only lowercases and converts spaces to underscores, so
// "Amber Valley", "amber valley" and "Amber_Valley" all map to the folder
// amber_valley. CreateZone's duplicate check compares DISPLAY names and so
// misses this, reaching os.Mkdir on a live zone's folder.
func ZoneFolderCollision(newName string, existing []string) string {
	folder := ZoneNameSanitize(newName)
	if folder == "" {
		return ""
	}
	for _, z := range existing {
		if ZoneNameSanitize(z) == folder {
			return z
		}
	}
	return ""
}
