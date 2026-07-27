package users

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	ErrIndexMissing        = errors.New(`index does not exist`)
	ErrUserFilesOldFormat  = errors.New(`user files are in old format of username.yaml`)
	ErrIndexVersionInvalid = errors.New(`version out of date.`)
	ErrSearchNameTooLong   = errors.New(`search name provided is too long`)
	ErrNotFound            = errors.New("user not found")
)

const (
	IndexVersion           = 3
	IndexLineTerminatorV1  = byte(10) // "\n"
	IndexRecordSizeV1      = 89
	IndexRecordSizeV3      = 185 // username[80] + userid(8) + charname[80] + mtime(8) + size(8) + newline
	FixedHeaderTotalLength = 100 // 99 bytes header content + 1 byte newline
)

// IndexMetaData holds header info that helps in reading the file.
type IndexMetaData struct {
	MetaDataSize uint64 // size of the metadata header (in bytes)
	IndexVersion uint64
	RecordCount  uint64
	RecordSize   uint64
	Checksum     uint64 // FNV-64 fingerprint of user directory contents; 0 when IndexChecksumEnabled is false
}

// IndexUserRecord represents one fixed-width record. Alongside the lookup
// fields it stores the active character name plus the mtime and size the
// user file had when it was indexed, so startup can tell unchanged files
// apart from changed ones without opening them.
type IndexUserRecord struct {
	UserID        int64
	Username      [80]byte
	CharacterName [80]byte
	FileModTime   int64 // UnixNano mtime of the user file when last indexed
	FileSize      int64 // size in bytes of the user file when last indexed
}

// UserIndex is the central struct that holds the index filename and methods
// to work with the index.
type UserIndex struct {
	mu            sync.RWMutex
	metaData      IndexMetaData
	highestUserId int
	Filename      string

	records    []IndexUserRecord
	byUsername map[string]int64
	byUserId   map[int64]string
}

var userIndex *UserIndex

// InitUserIndex creates and initializes the singleton UserIndex. Called once at startup.
func InitUserIndex() *UserIndex {
	filename := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, `users.idx`)
	userIndex = &UserIndex{Filename: filename}
	if userIndex.Exists() {
		userIndex.metaData = userIndex.getMetaDataFromFile()
		userIndex.loadRecords()
	}
	return userIndex
}

// GetUserIndex returns the singleton UserIndex, initializing it if needed.
func GetUserIndex() *UserIndex {
	if userIndex == nil {
		return InitUserIndex()
	}
	return userIndex
}

func (idx *UserIndex) Exists() bool {
	_, err := os.Stat(idx.Filename)
	return err == nil
}

func (idx *UserIndex) Delete() {
	if idx.Exists() {
		os.Remove(idx.Filename)
	}
}

// loadRecords bulk-reads all records from disk into memory and builds lookup maps.
func (idx *UserIndex) loadRecords() {
	idx.byUsername = make(map[string]int64, idx.metaData.RecordCount)
	idx.byUserId = make(map[int64]string, idx.metaData.RecordCount)
	idx.highestUserId = 0

	if idx.metaData.RecordCount == 0 {
		idx.records = nil
		return
	}

	// An index in an older record format is not readable - leave the maps
	// empty so SyncWithUserFiles falls back to a full rebuild.
	if idx.metaData.IndexVersion != IndexVersion || idx.metaData.RecordSize != IndexRecordSizeV3 {
		mudlog.Info("UserIndex", "info", "index format is outdated, a rebuild will recreate it", "path", idx.Filename, "version", idx.metaData.IndexVersion)
		idx.records = nil
		return
	}

	f, err := os.Open(idx.Filename)
	if err != nil {
		mudlog.Error("UserIndex", "error", "failed to open index file", "path", idx.Filename, "details", err)
		return
	}
	defer f.Close()

	dataSize := idx.metaData.RecordCount * idx.metaData.RecordSize
	buf := make([]byte, dataSize)
	if _, err := f.Seek(int64(idx.metaData.MetaDataSize), io.SeekStart); err != nil {
		mudlog.Error("UserIndex", "error", "failed to seek past index header", "path", idx.Filename, "details", err)
		return
	}
	if _, err := io.ReadFull(f, buf); err != nil {
		mudlog.Error("UserIndex", "error", "index file is truncated or corrupt, a rebuild will recreate it", "path", idx.Filename, "details", err)
		return
	}

	idx.records = make([]IndexUserRecord, idx.metaData.RecordCount)

	for i := uint64(0); i < idx.metaData.RecordCount; i++ {
		offset := i * idx.metaData.RecordSize
		rec := &idx.records[i]
		copy(rec.Username[:], buf[offset:offset+80])
		rec.UserID = int64(binary.LittleEndian.Uint64(buf[offset+80 : offset+88]))
		copy(rec.CharacterName[:], buf[offset+88:offset+168])
		rec.FileModTime = int64(binary.LittleEndian.Uint64(buf[offset+168 : offset+176]))
		rec.FileSize = int64(binary.LittleEndian.Uint64(buf[offset+176 : offset+184]))

		username := string(bytes.TrimRight(rec.Username[:], "\x00"))
		idx.byUsername[username] = rec.UserID
		idx.byUserId[rec.UserID] = username

		if int(rec.UserID) > idx.highestUserId {
			idx.highestUserId = int(rec.UserID)
		}
	}
}

// Create initializes a new empty index file with a header.
func (idx *UserIndex) Create() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.Delete()

	f, err := os.Create(idx.Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	idx.metaData = IndexMetaData{
		MetaDataSize: FixedHeaderTotalLength,
		IndexVersion: IndexVersion,
		RecordCount:  0,
		RecordSize:   IndexRecordSizeV3,
	}
	idx.highestUserId = 0
	idx.records = nil
	idx.byUsername = make(map[string]int64)
	idx.byUserId = make(map[int64]string)

	headerBytes, err := idx.metaData.Format()
	if err != nil {
		return err
	}
	if _, err := f.Write(headerBytes); err != nil {
		return err
	}

	return nil
}

// computeDirChecksum returns a FNV-64 fingerprint over the name, mtime
// (nanoseconds), and size of every qualifying user YAML file in basePath.
// The file count is also folded in so that deletions change the checksum.
func computeDirChecksum(basePath string) (uint64, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return 0, err
	}

	h := fnv.New64a()
	var count uint64
	var buf [24]byte

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, `.yaml`) || strings.HasSuffix(name, `.alts.yaml`) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return 0, err
		}
		h.Write([]byte(name))
		binary.LittleEndian.PutUint64(buf[0:8], uint64(info.ModTime().UnixNano()))
		binary.LittleEndian.PutUint64(buf[8:16], uint64(info.Size()))
		h.Write(buf[:16])
		count++
	}

	binary.LittleEndian.PutUint64(buf[0:8], count)
	h.Write(buf[:8])

	return h.Sum64(), nil
}

// IsUpToDate returns true if the index file exists, has the current version,
// actually loaded every record its header claims, and its stored FNV-64
// checksum matches the current state of the user directory.
func (idx *UserIndex) IsUpToDate() bool {
	basePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`)
	return idx.isUpToDateForDir(basePath)
}

// isUpToDateForDir is IsUpToDate parameterized by the directory to compare
// against, so tests can point it at a synthetic users directory.
func (idx *UserIndex) isUpToDateForDir(basePath string) bool {
	if !idx.Exists() {
		return false
	}
	if idx.metaData.IndexVersion != IndexVersion {
		return false
	}

	// A truncated or unreadable records section leaves fewer records in
	// memory than the header claims. Such an index must never be trusted:
	// with empty maps, GetUniqueUserId would start handing out userids that
	// already belong to existing user files.
	if uint64(len(idx.records)) != idx.metaData.RecordCount {
		return false
	}

	current, err := computeDirChecksum(basePath)
	if err != nil {
		return false
	}
	return idx.metaData.Checksum == current
}

// Rebuild recreates the index from the user files on disk. It runs a
// lightweight scan (userid and username only) instead of fully loading every
// user record, then writes the whole index in one atomic pass (temp file +
// rename) with a single sync, instead of appending and syncing once per
// user. The directory checksum is folded into that same write so IsUpToDate
// can detect stale indexes on the next startup.
func (idx *UserIndex) Rebuild() error {
	return idx.RebuildFromScan(ScanUserFiles())
}

// RebuildFromScan is Rebuild fed by an existing scan, so startup can share
// one scan between the user index and the character index.
func (idx *UserIndex) RebuildFromScan(scan []UserFileScan) error {
	basePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`)
	checksum, err := computeDirChecksum(basePath)
	if err != nil {
		return fmt.Errorf("checksum compute failed: %w", err)
	}
	return idx.applyScan(scan, checksum)
}

// scanRecord converts one scan result into a fixed-width index record.
func scanRecord(s UserFileScan) IndexUserRecord {
	rec := IndexUserRecord{
		UserID:      int64(s.UserId),
		FileModTime: s.FileModTime,
		FileSize:    s.FileSize,
	}
	copy(rec.Username[:], strings.ToLower(s.Username))
	copy(rec.CharacterName[:], s.CharacterName)
	return rec
}

// applyScan replaces the in-memory records and lookup maps with the scan
// results, then writes the complete index to disk once.
func (idx *UserIndex) applyScan(scan []UserFileScan, checksum uint64) error {
	records := make([]IndexUserRecord, 0, len(scan))
	for _, s := range scan {
		records = append(records, scanRecord(s))
	}
	return idx.applyRecords(records, checksum)
}

// applyRecords replaces the in-memory records and lookup maps, then writes
// the complete index to disk once.
func (idx *UserIndex) applyRecords(records []IndexUserRecord, checksum uint64) error {

	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.metaData = IndexMetaData{
		MetaDataSize: FixedHeaderTotalLength,
		IndexVersion: IndexVersion,
		RecordCount:  uint64(len(records)),
		RecordSize:   IndexRecordSizeV3,
		Checksum:     checksum,
	}

	idx.records = records
	idx.byUsername = make(map[string]int64, len(records))
	idx.byUserId = make(map[int64]string, len(records))
	idx.highestUserId = 0

	for _, rec := range records {
		username := string(bytes.TrimRight(rec.Username[:], "\x00"))
		idx.byUsername[username] = rec.UserID
		idx.byUserId[rec.UserID] = username
		if int(rec.UserID) > idx.highestUserId {
			idx.highestUserId = int(rec.UserID)
		}
	}

	// The in-memory state is updated even if the disk write fails - the
	// running process must trust what was just scanned, not a stale file.
	if err := idx.writeCompleteIndex(records); err != nil {
		return fmt.Errorf("index write failed: %w", err)
	}

	return nil
}

func (idx *UserIndex) GetMetaData() IndexMetaData {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.metaData
}

func (idx *UserIndex) GetHighestUserId() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.highestUserId
}

// ForEachRecord iterates over all index records, calling fn for each.
// Returning false from fn stops iteration.
func (idx *UserIndex) ForEachRecord(fn func(rec IndexUserRecord) bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	for _, rec := range idx.records {
		if !fn(rec) {
			return
		}
	}
}

// FindByUsername searches the index for a username and returns its userId.
func (idx *UserIndex) FindByUsername(username string) (int, bool) {
	if len(username) > 80 {
		return 0, false
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	userId, ok := idx.byUsername[strings.ToLower(username)]
	if !ok {
		return 0, false
	}
	return int(userId), true
}

// FindByUserId searches for a user record matching the provided userId.
func (idx *UserIndex) FindByUserId(userId int) (string, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	username, ok := idx.byUserId[int64(userId)]
	if !ok {
		return "", false
	}
	return username, true
}

func (idx *UserIndex) getMetaDataFromFile() IndexMetaData {
	f, err := os.Open(idx.Filename)
	if err != nil {
		return IndexMetaData{}
	}
	defer f.Close()

	header := make([]byte, FixedHeaderTotalLength)
	if _, err := io.ReadFull(f, header); err != nil {
		return IndexMetaData{}
	}

	var meta IndexMetaData
	meta.MetaDataSize = uint64(len(header))
	headerContent := strings.TrimSpace(string(header[:FixedHeaderTotalLength-1]))
	n, _ := fmt.Sscanf(headerContent, "VERSION=%d,RECORDCOUNT=%d,RECORDSIZE=%d,CHECKSUM=%d", &meta.IndexVersion, &meta.RecordCount, &meta.RecordSize, &meta.Checksum)
	if n < 3 {
		mudlog.Error("UserIndex", "error", "index header is unparseable, a rebuild will recreate it", "path", idx.Filename)
		return IndexMetaData{}
	}

	return meta
}

// AddUser appends a new record to the index file and updates the header.
func (idx *UserIndex) AddUser(userId int, username string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	username = strings.ToLower(username)

	newRecord := IndexUserRecord{
		UserID: int64(userId),
	}
	copy(newRecord.Username[:], username)

	f, err := os.OpenFile(idx.Filename, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("error seeking to file end: %w", err)
	}

	// The character name, mtime, and size are left zeroed here - the user
	// file usually doesn't exist yet when a user is first registered. The
	// zero mtime never matches the real file, so the next startup sync
	// re-parses that one file and completes the record.
	recBuf := encodeIndexRecord(newRecord)
	if _, err := f.Write(recBuf[:]); err != nil {
		return fmt.Errorf("error writing record: %w", err)
	}

	if userId > idx.highestUserId {
		idx.highestUserId = userId
	}

	idx.metaData.RecordCount++

	newHeaderBytes, err := idx.metaData.Format()
	if err != nil {
		return fmt.Errorf("error formatting header: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("error seeking to beginning: %w", err)
	}
	if _, err := f.Write(newHeaderBytes); err != nil {
		return fmt.Errorf("error writing updated header: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("error syncing file: %w", err)
	}

	idx.records = append(idx.records, newRecord)
	idx.byUsername[username] = int64(userId)
	idx.byUserId[int64(userId)] = username

	return nil
}

// RemoveByUsername removes the first record matching the username and rewrites the index.
func (idx *UserIndex) RemoveByUsername(username string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	username = strings.ToLower(username)

	if _, ok := idx.byUsername[username]; !ok {
		return ErrNotFound
	}

	newRecords := make([]IndexUserRecord, 0, len(idx.records)-1)
	removed := false

	for _, rec := range idx.records {
		if !removed {
			recUser := string(bytes.TrimRight(rec.Username[:], "\x00"))
			if recUser == username {
				removed = true
				delete(idx.byUsername, username)
				delete(idx.byUserId, rec.UserID)
				continue
			}
		}
		newRecords = append(newRecords, rec)
	}

	idx.records = newRecords
	idx.metaData.RecordCount = uint64(len(newRecords))

	idx.highestUserId = 0
	for _, rec := range idx.records {
		if int(rec.UserID) > idx.highestUserId {
			idx.highestUserId = int(rec.UserID)
		}
	}

	return idx.writeCompleteIndex(newRecords)
}

// Format formats the metadata header as a fixed-width string.
// The header (without newline) is exactly 99 bytes.
func (m IndexMetaData) Format() ([]byte, error) {
	headerContent := fmt.Sprintf("VERSION=%d,RECORDCOUNT=%d,RECORDSIZE=%d,CHECKSUM=%d", m.IndexVersion, m.RecordCount, m.RecordSize, m.Checksum)
	if len(headerContent) > FixedHeaderTotalLength-1 {
		return nil, fmt.Errorf("header content too long: %d bytes", len(headerContent))
	}
	padded := headerContent + strings.Repeat(" ", FixedHeaderTotalLength-1-len(headerContent))
	return []byte(padded + string(IndexLineTerminatorV1)), nil
}

// SyncWithUserFiles brings the index in line with the user files on disk,
// parsing only files that are new or changed since they were last indexed
// and dropping records whose files are gone. Unchanged files are never
// opened. Returns how many records had to be parsed or dropped; when that
// is zero the index file is not rewritten at all.
func (idx *UserIndex) SyncWithUserFiles() (int, error) {
	basePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`)
	return idx.syncWithDir(basePath)
}

// syncWithDir is SyncWithUserFiles parameterized by the directory to sync
// against, so tests can point it at a synthetic users directory.
func (idx *UserIndex) syncWithDir(basePath string) (int, error) {

	// An index that is missing, in an older format, or that did not load
	// every record its header claims cannot be trusted for incremental
	// work - fall back to a full scan of every user file.
	idx.mu.RLock()
	trustworthy := idx.metaData.IndexVersion == IndexVersion &&
		idx.metaData.RecordSize == IndexRecordSizeV3 &&
		uint64(len(idx.records)) == idx.metaData.RecordCount
	oldRecords := idx.records
	idx.mu.RUnlock()

	if !idx.Exists() {
		trustworthy = false
	}

	if !trustworthy {
		scan := scanUserFilesInDir(basePath)
		checksum, err := computeDirChecksum(basePath)
		if err != nil {
			return 0, fmt.Errorf("checksum compute failed: %w", err)
		}
		return len(scan), idx.applyScan(scan, checksum)
	}

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return 0, fmt.Errorf("users directory unreadable: %w", err)
	}

	onDisk := make(map[string]os.FileInfo, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, `.yaml`) || strings.HasSuffix(name, `.alts.yaml`) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return 0, fmt.Errorf("stat failed for %s: %w", name, err)
		}
		onDisk[name] = info
	}

	// Records are matched to files by the <userid>.yaml naming convention.
	// A user file in the old username.yaml naming never matches a record,
	// so it is re-parsed every startup - run the format migration to avoid
	// that cost.
	changed := 0
	kept := make([]IndexUserRecord, 0, len(oldRecords))
	reparse := []string{}
	claimed := make(map[string]bool, len(oldRecords))

	for _, rec := range oldRecords {
		fileName := strconv.FormatInt(rec.UserID, 10) + `.yaml`
		info, ok := onDisk[fileName]
		if !ok {
			changed++
			continue
		}
		claimed[fileName] = true
		if info.ModTime().UnixNano() == rec.FileModTime && info.Size() == rec.FileSize {
			kept = append(kept, rec)
			continue
		}
		reparse = append(reparse, fileName)
		changed++
	}

	for name := range onDisk {
		if !claimed[name] {
			reparse = append(reparse, name)
			changed++
		}
	}

	if changed == 0 {
		return 0, nil
	}

	for _, name := range reparse {
		if s, ok := scanUserFile(filepath.Join(basePath, name), onDisk[name]); ok {
			kept = append(kept, scanRecord(s))
		}
	}

	checksum, err := computeDirChecksum(basePath)
	if err != nil {
		return 0, fmt.Errorf("checksum compute failed: %w", err)
	}

	return changed, idx.applyRecords(kept, checksum)
}

// encodeIndexRecord serializes one record into its fixed-width on-disk form.
func encodeIndexRecord(rec IndexUserRecord) [IndexRecordSizeV3]byte {
	var recBuf [IndexRecordSizeV3]byte
	copy(recBuf[:80], rec.Username[:])
	binary.LittleEndian.PutUint64(recBuf[80:88], uint64(rec.UserID))
	copy(recBuf[88:168], rec.CharacterName[:])
	binary.LittleEndian.PutUint64(recBuf[168:176], uint64(rec.FileModTime))
	binary.LittleEndian.PutUint64(recBuf[176:184], uint64(rec.FileSize))
	recBuf[184] = IndexLineTerminatorV1
	return recBuf
}

// writeCompleteIndex writes metadata and all records atomically via temp file + rename.
func (idx *UserIndex) writeCompleteIndex(records []IndexUserRecord) error {
	tmpFile := idx.Filename + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	writeErr := func() error {
		headerBytes, err := idx.metaData.Format()
		if err != nil {
			return err
		}

		buf := make([]byte, 0, len(headerBytes)+len(records)*IndexRecordSizeV3)
		buf = append(buf, headerBytes...)

		for _, rec := range records {
			recBuf := encodeIndexRecord(rec)
			buf = append(buf, recBuf[:]...)
		}

		if _, err := f.Write(buf); err != nil {
			return err
		}

		return f.Sync()
	}()

	if closeErr := f.Close(); writeErr == nil {
		writeErr = closeErr
	}

	if writeErr != nil {
		os.Remove(tmpFile)
		return writeErr
	}

	return os.Rename(tmpFile, idx.Filename)
}
