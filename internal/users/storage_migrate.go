package users

// MigrateStorageSlots converts the legacy Storage.Items flat list into the
// new Storage.Slots shape. It is idempotent: if Items is already empty (the
// migration has run before), it returns false without touching Slots.
//
// Called from LoadUser after YAML deserialisation, alongside the character
// migration calls.
//
// Returns true if any data was migrated (caller should mark user dirty).
func (s *Storage) MigrateStorageSlots() bool {
	if len(s.Items) == 0 {
		return false
	}

	for _, itm := range s.Items {
		s.AddItem(itm) // AddItem handles stack folding
	}

	// Clear the legacy field so the next save writes only Slots.
	s.Items = nil

	return true
}
