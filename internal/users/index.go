package users

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
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
	IndexVersion           = 1
	IndexLineTerminatorV1  = byte(10) // "\n"
	IndexRecordSizeV1      = 89
	FixedHeaderTotalLength = 100 // 99 bytes header content + 1 byte newline
)

// IndexMetaData holds header info that helps in reading the file.
type IndexMetaData struct {
	MetaDataSize uint64 // size of the metadata header (in bytes)
	IndexVersion uint64
	RecordCount  uint64
	RecordSize   uint64
}

// IndexUserRecord represents one fixed-width record.
type IndexUserRecord struct {
	UserID   int64
	Username [80]byte
}

// UserIndex is the central struct that holds the index filename and methods
// to work with the index.
type UserIndex struct {
	metaData      IndexMetaData
	highestUserId int
	Filename      string
}

// NewUserIndex creates a new instance of UserIndex using the configured file path.
func NewUserIndex() *UserIndex {
	filename := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, `users.idx`)
	idx := &UserIndex{Filename: filename}
	if idx.Exists() {
		idx.metaData = idx.getMetaDataFromFile()
		idx.calculateHighestUserId()
	}
	return idx
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

// Writes a new index header then processes all user records to write a new index
func (idx *UserIndex) Create() error {

	idx.Delete()

	f, err := os.Create(idx.Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Reset metadata.
	idx.metaData = IndexMetaData{
		MetaDataSize: FixedHeaderTotalLength,
		IndexVersion: IndexVersion,
		RecordCount:  0,
		RecordSize:   IndexRecordSizeV1,
	}

	headerBytes, err := idx.metaData.Format()
	if err != nil {
		return err
	}
	if _, err := f.Write(headerBytes); err != nil {
		return err
	}

	return nil
}

func (idx *UserIndex) GetMetaData() IndexMetaData {
	return idx.metaData
}

func (idx *UserIndex) GetHighestUserId() int {
	return idx.highestUserId
}

func (idx *UserIndex) calculateHighestUserId() {

	f, err := os.Open(idx.Filename)
	if err != nil {
		return
	}
	defer f.Close()

	for i := uint64(0); i < idx.metaData.RecordCount; i++ {
		offset := int64(idx.metaData.MetaDataSize) + int64(i*idx.metaData.RecordSize)
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return
		}

		var recUsername [80]byte
		if n, err := io.ReadFull(f, recUsername[:]); err != nil || n != 80 {
			return
		}

		var userId int64
		if err := binary.Read(f, binary.LittleEndian, &userId); err != nil {
			return
		}

		if int(userId) > idx.highestUserId {
			idx.highestUserId = int(userId)
		}
	}

}

// FindByUsername opens the index file, reads its header, then iterates over
func (idx *UserIndex) FindByUsername(username string) (int64, bool) {
	if len(username) > 80 {
		return 0, false
	}

	f, err := os.Open(idx.Filename)
	if err != nil {
		return 0, false
	}
	defer f.Close()

	username = strings.ToLower(username)

	for i := uint64(0); i < idx.metaData.RecordCount; i++ {
		offset := int64(idx.metaData.MetaDataSize) + int64(i*idx.metaData.RecordSize)
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return 0, false
		}

		var recUsername [80]byte
		if n, err := io.ReadFull(f, recUsername[:]); err != nil || n != 80 {
			return 0, false
		}

		var userId int64
		if err := binary.Read(f, binary.LittleEndian, &userId); err != nil {
			return 0, false
		}

		term := make([]byte, 1)
		if _, err := f.Read(term); err != nil {
			return 0, false
		}
		if term[0] != IndexLineTerminatorV1 {
			return 0, false
		}

		// Compare only the first len(username) characters.
		if len(username) < 80 && recUsername[len(username)] != 0 {
			continue
		}
		match := true
		for j := 0; j < len(username); j++ {
			if username[j] != recUsername[j] {
				match = false
				break
			}
		}
		if match {
			return userId, true
		}
	}
	return 0, false
}

// FindByUserId searches for a user record matching the provided userId.
// If found, it returns the corresponding username.
func (idx *UserIndex) FindByUserId(userId int64) (string, bool) {
	f, err := os.Open(idx.Filename)
	if err != nil {
		return "", false
	}
	defer f.Close()

	for i := uint64(0); i < idx.metaData.RecordCount; i++ {
		offset := int64(idx.metaData.MetaDataSize) + int64(i*idx.metaData.RecordSize)
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return "", false
		}

		var recUsername [80]byte
		if n, err := io.ReadFull(f, recUsername[:]); err != nil || n != 80 {
			return "", false
		}

		var recUserId int64
		if err := binary.Read(f, binary.LittleEndian, &recUserId); err != nil {
			return "", false
		}

		term := make([]byte, 1)
		if _, err := f.Read(term); err != nil {
			return "", false
		}
		if term[0] != IndexLineTerminatorV1 {
			return "", false
		}

		if recUserId == userId {
			username := string(bytes.TrimRight(recUsername[:], "\x00"))
			return username, true
		}
	}
	return "", false
}

func (idx *UserIndex) getMetaDataFromFile() IndexMetaData {

	f, err := os.Open(idx.Filename)
	if err != nil {
		return IndexMetaData{}
	}
	defer f.Close()

	headerBytes, err := idx.readFixedHeader(f)
	if err != nil {
		return IndexMetaData{}
	}

	var meta IndexMetaData
	meta.MetaDataSize = uint64(len(headerBytes))
	headerContent := strings.TrimSpace(string(headerBytes[:FixedHeaderTotalLength-1]))
	fmt.Sscanf(headerContent, "VERSION=%d,RECORDCOUNT=%d,RECORDSIZE=%d", &meta.IndexVersion, &meta.RecordCount, &meta.RecordSize)

	return meta
}

// AppendUserRecord appends a new record to the index file and updates the header.
func (idx *UserIndex) AddUser(userId int, username string) error {

	// We lowercase username so that we can ensure uniqueness
	username = strings.ToLower(username)

	// Create the new record
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
	if _, err := f.Write(newRecord.Username[:]); err != nil {
		return fmt.Errorf("error writing username: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, newRecord.UserID); err != nil {
		return fmt.Errorf("error writing userId: %w", err)
	}
	if _, err := f.Write([]byte{IndexLineTerminatorV1}); err != nil {
		return fmt.Errorf("error writing record terminator: %w", err)
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
	return nil
}

// RemoveUserRecordByUsername removes the first record that matches the provided username,
// updates the header, and rewrites the file.
func (idx *UserIndex) RemoveByUsername(username string) error {

	f, err := os.Open(idx.Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	username = strings.ToLower(username)

	records := make([]IndexUserRecord, 0, idx.metaData.RecordCount)
	recordFound := false

	for i := uint64(0); i < idx.metaData.RecordCount; i++ {
		offset := int64(idx.metaData.MetaDataSize) + int64(i*idx.metaData.RecordSize)
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return fmt.Errorf("seek error: %w", err)
		}

		var rec IndexUserRecord
		if n, err := f.Read(rec.Username[:]); err != nil || n != 80 {
			return fmt.Errorf("error reading username: %w", err)
		}
		if err := binary.Read(f, binary.LittleEndian, &rec.UserID); err != nil {
			return fmt.Errorf("error reading userId: %w", err)
		}
		term := make([]byte, 1)
		if _, err := f.Read(term); err != nil {
			return fmt.Errorf("error reading record terminator: %w", err)
		}
		if term[0] != IndexLineTerminatorV1 {
			return fmt.Errorf("invalid record terminator")
		}

		recUserStr := string(bytes.TrimRight(rec.Username[:], "\x00"))
		if recUserStr == username && !recordFound {
			recordFound = true
			continue // skip this record
		}
		records = append(records, rec)
	}

	if !recordFound {
		return ErrNotFound
	}

	idx.metaData.RecordCount = uint64(len(records))

	return idx.writeCompleteIndex(records)
}

// formatFixedHeader formats the metadata header as a fixed-width string.
// The header (without newline) is exactly 99 bytes.
func (m IndexMetaData) Format() ([]byte, error) {
	headerContent := fmt.Sprintf("VERSION=%d,RECORDCOUNT=%d,RECORDSIZE=%d", m.IndexVersion, m.RecordCount, m.RecordSize)
	if len(headerContent) > FixedHeaderTotalLength-1 {
		return nil, fmt.Errorf("header content too long: %d bytes", len(headerContent))
	}
	padded := headerContent + strings.Repeat(" ", FixedHeaderTotalLength-1-len(headerContent))
	return []byte(padded + string(IndexLineTerminatorV1)), nil
}

// readFixedHeader reads exactly 100 bytes (the fixed header) from the provided reader.
func (idx *UserIndex) readFixedHeader(r io.Reader) ([]byte, error) {
	header := make([]byte, FixedHeaderTotalLength)
	n, err := io.ReadFull(r, header)
	if err != nil {
		return nil, err
	}
	if n != FixedHeaderTotalLength {
		return nil, fmt.Errorf("expected %d bytes for header, got %d", FixedHeaderTotalLength, n)
	}
	return header, nil
}

// writeIndex writes the metadata header and all user records into the index file.
func (idx *UserIndex) writeCompleteIndex(records []IndexUserRecord) error {
	f, err := os.Create(idx.Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	headerBytes, err := idx.metaData.Format()
	if err != nil {
		return err
	}
	if _, err := f.Write(headerBytes); err != nil {
		return err
	}

	for _, rec := range records {
		if _, err := f.Write(rec.Username[:]); err != nil {
			return err
		}
		if err := binary.Write(f, binary.LittleEndian, rec.UserID); err != nil {
			return err
		}
		if _, err := f.Write([]byte{IndexLineTerminatorV1}); err != nil {
			return err
		}
	}
	return nil
}
