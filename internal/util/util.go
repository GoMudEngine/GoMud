package util

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"crypto/md5"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/mattn/go-runewidth"
)

var (
	turnCount    uint64 = 0
	roundCount   uint64 = RoundCountMinimum
	timeTrackers        = map[string]*Accumulator{}
	serverAddr   string = `Unknown`

	strippablePrepositions = []string{
		`onto`,
		`into`,
		`over`,
		`to`,
		`toward`,
		`towards`,
		`from`,
		`in`,
		`under`,
		`upon`,
		`with`,
		`the`, // also strip this because it's unnecessary
		`my`,  // also strip this because it's unnecessary
	}

	colorShortTagRegex = regexp.MustCompile(`\{(\d*)(?::)?(\d*)?\}`)

	// One CJK character, punctuation, and symbol is one word.
	// \p{Han}: chinese
	// \p{Hiragana}: Japanese
	// \p{Katakana}: Japanese
	// \p{Hangul}: Korean
	// \w: alphanumeric
	// \p{P}: punctuation
	// \p{S}: symbol
	wordRegex        = regexp.MustCompile(`(</?ansi[^>]*>|[\p{Han}\p{Hiragana}\p{Katakana}\p{Hangul}]|\w+|[\p{P}\p{S}\s]+)`)
	punctuationRegex = regexp.MustCompile(`[\p{P}]+`)
	ansiTagRegex = regexp.MustCompile(`</?ansi[^>]*>`)

	mudLock = sync.RWMutex{}
)

const (
	// start at 1314000 (approx. 4 years in the future) to avoid complexities of
	// delta comparisons and to allow for date adjustments.
	RoundCountMinimum  = 1314000
	RoundCountFilename = `.roundcount`
)

// Mutex lock intended for synchronizing at a high level between
// components that may asyncronously access game data
func LockMud() {
	mudLock.Lock()
}

func UnlockMud() {
	mudLock.Unlock()
}

func RLockMud() {
	mudLock.RLock()
}

func RUnlockMud() {
	mudLock.RUnlock()
}

//
// End Mutex
//

func SetServerAddress(addr string) {
	serverAddr = addr
}

func GetServerAddress() string {
	return serverAddr
}

func SetRoundCount(newRoundCount uint64) {
	roundCount = newRoundCount
}

func IncrementTurnCount() uint64 {
	turnCount++
	return turnCount
}

func GetTurnCount() uint64 {
	return turnCount
}

func IncrementRoundCount() uint64 {
	roundCount++
	return roundCount
}

func GetRoundCount() uint64 {
	return roundCount
}

func TrackTime(name string, timePassed float64) {
	if _, ok := timeTrackers[name]; !ok {
		timeTrackers[name] = &Accumulator{
			Name:  name,
			Start: time.Now()}
	}
	timeTrackers[name].Record(timePassed)
}

func GetTimeTrackers() []Accumulator {

	result := []Accumulator{}
	for _, t := range timeTrackers {
		result = append(result, *t)
	}

	return result
}

type Accumulator struct {
	Name    string
	Total   float64
	Lowest  float64
	Highest float64
	Count   float64
	Start   time.Time
}

func (t *Accumulator) Stats() (lowest float64, highest float64, average float64, count float64) {
	return t.Lowest, t.Highest, t.Average(), t.Count
}

func (t *Accumulator) Average() float64 {
	return t.Total / t.Count
}

func (t *Accumulator) Record(nextValue float64) {
	t.Count++
	t.Total += nextValue
	if nextValue < t.Lowest || t.Lowest == 0 {
		t.Lowest = nextValue
	}
	if nextValue > t.Highest {
		t.Highest = nextValue
	}
}

func Rand(maxInt int) int {
	if maxInt < 1 {
		return 0
	}
	return rand.Intn(maxInt)
}

func LogRoll(name string, rollResult int, targetNumber int) {
	success := rollResult < targetNumber
	mudlog.Debug(`Rand Result`, `Name`, name, `Result`, fmt.Sprintf(`%d < %d`, rollResult, targetNumber), `Success`, success)
}

// VisibleWidth returns the display width of a string, ignoring <ansi> markup tags.
func VisibleWidth(s string) int {
	return runewidth.StringWidth(ansiTagRegex.ReplaceAllString(s, ``))
}

func SplitString(input string, lineWidth int) []string {
	var result []string
	var currentLine string
	currentLen := 0

	parts := strings.Split(input, "\n")

	for _, textLine := range parts {
		words := wordRegex.FindAllString(textLine, -1)

		l := len(words)

		skip := false
		for idx, word := range words {
			if skip {
				skip = false
				continue
			}

			wordLen := VisibleWidth(word)

			if idx < l-1 && punctuationRegex.MatchString(words[idx+1]) {
				wordLen += VisibleWidth(words[idx+1])
				word += words[idx+1]
				skip = true
			} else {
				skip = false
			}

			if wordLen > lineWidth {
				result = append(result, word)
				continue
			}

			if currentLen+wordLen > lineWidth {
				if currentLine != "" {
					result = append(result, strings.TrimRight(currentLine, " "))
				}
				// clear spaces at the beginning of the line
				currentLine = strings.TrimLeft(word, " ")
				currentLen = VisibleWidth(currentLine)
			} else {
				currentLine += word
				currentLen += wordLen
			}
		}

		if currentLine != "" {
			result = append(result, currentLine)
			currentLine = ""
			currentLen = 0
		}
	}

	return result
}

// Splits a string by adding line breaks at the end of each line
func SplitStringNL(input string, lineWidth int, nlPrefix ...string) string {
	lines := SplitString(input, lineWidth)

	linePrefix := ""
	if len(nlPrefix) > 0 {
		linePrefix = nlPrefix[0]
	}

	return strings.Join(lines, term.CRLFStr+linePrefix)
}

func SplitButRespectQuotes(s string) []string {

	// This regex matches either a quoted string (with either single or double quotes) or a non-space sequence.
	// For example, for the input: `hello "my name" is 'Sammy'`
	// It matches: [`hello", ""my name"", "is", "'Sammy'`]
	re := regexp.MustCompile(`("[^"]*")|('[^']*')|\S+`)
	matches := re.FindAllString(s, -1)
	finalMatches := make([]string, 0, 1)

	// Remove quotes around the matches, if they exist
	for _, match := range matches {

		match = strings.TrimSpace(match)

		if strings.HasPrefix(match, `"`) && strings.HasSuffix(match, `"`) ||
			strings.HasPrefix(match, `'`) && strings.HasSuffix(match, `'`) {
			str := strings.TrimSpace(match[1 : len(match)-1])
			finalMatches = append(finalMatches, str)
		} else {
			finalMatches = append(finalMatches, match)
		}
	}

	return finalMatches
}

// GetMatchNumber accepts an input and extracts a match number.
// Supports three formats:
//   - "N.item"   (diku-style) → ("item", N)
//   - "all.item"              → ("item", -1)  sentinel for "all matches"
//   - "item#N"   (existing)   → ("item", N)
//   - plain "item"            → ("item", 1)
func GetMatchNumber(input string) (string, int) {
	input = strings.TrimSpace(strings.ToLower(input))

	// Check for N.item or all.item prefix
	if dotIdx := strings.IndexByte(input, '.'); dotIdx > 0 {
		prefix := input[:dotIdx]
		rest := input[dotIdx+1:]
		if len(rest) > 0 {
			if prefix == "all" {
				return rest, -1
			}
			if n, err := strconv.Atoi(prefix); err == nil && n >= 1 {
				return rest, n
			}
		}
	}

	// Check for item#N suffix (existing logic)
	if strings.Contains(input, "#") {
		parts := strings.Split(input, "#")
		input = parts[0]
		inputNumber, _ := strconv.Atoi(strings.Join(parts[1:], "#"))
		if inputNumber < 1 {
			inputNumber = 1
		}
		return input, inputNumber
	}

	return input, 1
}

func FindMatchIn(searchName string, items ...string) (match string, closeMatch string) {

	if searchName == `` {
		return ``, `` // No match
	}

	searchName, searchNumber := GetMatchNumber(searchName)

	var matchCt int = 0
	var closeMatchCt int = 0

	for _, i := range items {

		part, full := stringMatch(searchName, i, false)

		if part {
			closeMatchCt++
			if closeMatchCt == searchNumber {
				closeMatch = i
			}
		}

		if full {
			matchCt++
			if matchCt == searchNumber {
				match = i
				break
			}
		}

	}

	// If no "starts with" or "exact" matches are found, try and find the first item that contain the supplied name
	// Note: Can't have an exact match if there was never a close match
	if len(closeMatch) == 0 {
		closeMatchCt = 0
		for _, i := range items {
			part, _ := stringMatch(searchName, i, true)

			if part {
				closeMatchCt++
				if closeMatchCt == searchNumber {
					closeMatch = i
					break
				}
			}

		}

	}

	return match, closeMatch
}

// Searches for a partial or full match of a string
// If allowContains is true, the match can appear anywhere in the string.
// Otherwise it must start with the searchFor string
func stringMatch(searchFor string, searchIn string, allowContains bool) (partialMatch bool, fullMatch bool) {

	searchFor = strings.ToLower(searchFor)
	searchIn = strings.ToLower(searchIn)

	if allowContains {
		if strings.Contains(searchIn, searchFor) {
			if searchIn == searchFor {
				return true, true
			}
			return true, false
		}
	}

	if strings.HasPrefix(searchIn, searchFor) {
		if searchIn == searchFor {
			return true, true
		}
		return true, false
	}

	return false, false
}

func Hash(input string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(input)))
}

func HashBytes(input []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(input))
}

func Md5(input string) string {
	hasher := md5.New()
	hasher.Write([]byte([]byte(input)))
	hashInBytes := hasher.Sum(nil)
	return hex.EncodeToString(hashInBytes)
}

func Md5Bytes(input []byte) []byte {
	hasher := md5.New()
	hasher.Write([]byte(input))
	return hasher.Sum(nil)
}

func GetLockSequence(lockIdentifier string, difficulty int, seed string) string {

	// A lock sequence is a sequence of UP or DOWN commands that must be entered to unlock a lock
	// The difficulty is how many commands are in the sequence
	// First generate a Md5Bytes() hash and then use the first N bytes to generate the sequence
	// The sequence is generated by taking the first N bytes of the hash and then converting each byte to a number
	// If the number is even, the command is UP, if it's odd, the command is DOWN
	// The sequence is then returned as a string of U's and D's

	// Clamp difficulty between [2..32]
	if difficulty < 2 {
		difficulty = 2
	} else if difficulty > 32 {
		difficulty = 32
	}

	// Generate the hash
	hashInput := strings.ToLower(lockIdentifier + seed)
	hash := Md5Bytes([]byte(hashInput))
	for len(hash) < difficulty {
		hash = append(hash, Md5Bytes([]byte(hashInput+strconv.Itoa(len(hash))))...)
	}

	// Pre-allocate a slice of bytes
	seq := make([]byte, difficulty)
	for i := 0; i < difficulty; i++ {
		if hash[i]%2 == 0 {
			seq[i] = 'U'
		} else {
			seq[i] = 'D'
		}
	}

	// Convert once to string
	return string(seq)
}

func Compress(input []byte) []byte {
	var b bytes.Buffer
	// Create a new gzip writer
	gz := gzip.NewWriter(&b)
	// Write the input data to the gzip writer
	if _, err := gz.Write(input); err != nil {
		return []byte{}
	}
	if err := gz.Close(); err != nil {
		return []byte{}
	}
	return b.Bytes()
}

func Decompress(input []byte) []byte {

	// Create a buffer to read from the compressed data
	b := bytes.NewBuffer(input)
	// Create a new gzip reader
	gr, err := gzip.NewReader(b)
	if err != nil {
		return []byte{}
	}
	defer gr.Close()

	// Read the uncompressed data from the gzip reader
	uncompressedData, err := io.ReadAll(gr)
	if err != nil {
		return []byte{}
	}

	return uncompressedData
}

func Encode(blobdata []byte) string {
	// base64 encode the bytes
	return base64.StdEncoding.EncodeToString(blobdata)
}

func Decode(base64str string) []byte {
	// base64 encode the bytes
	b, _ := base64.StdEncoding.DecodeString(base64str)
	return b
}

func GetMyIP() string {

	url := `https://api.ipify.org/?format=txt`

	resp, err := http.Get(url)
	if err != nil {
		return err.Error()
	}

	defer resp.Body.Close()
	// handle the error if there is one
	if err != nil {
		return err.Error()
	}

	// do this now so it won't be forgotten
	defer resp.Body.Close()
	// reads html as a slice of bytes
	html, err := io.ReadAll(resp.Body)
	if err != nil {
		return err.Error()
	}

	// show the HTML code as a string %s
	return string(html)
}

func ProgressBar(complete float64, maxBarSize int, barParts ...string) (fullBar string, emptyBar string) {
	fullBarPiece := `█`
	emptyBarPiece := `░`

	if len(barParts) >= 2 {
		fullBarPiece = barParts[0]
		emptyBarPiece = barParts[1]
	}

	fullBars := int(math.Floor(float64(maxBarSize) * complete))
	return strings.Repeat(fullBarPiece, fullBars), strings.Repeat(emptyBarPiece, maxBarSize-fullBars)
}

// Returns X dice rolled with Y sides
func RollDice(dice int, sides int) int {
	var total int

	invert := dice < 0

	if invert {
		dice *= -1
	}

	if sides < 0 {
		sides *= -1
	}

	for i := 0; i < dice; i++ {
		total += Rand(sides) + 1
	}

	if invert {
		return total * -1
	}

	return total
}

// Gets the specifics of the item damage
// Format:
// 2@1d3+2#1,2,3
func ParseDiceRoll(dRoll string) (attacks int, dCount int, dSides int, bonus int, buffOnCrit []int) {

	attacks = 1

	var dice []string

	// After # is a list of buffId's separated by commas
	if strings.Contains(dRoll, `#`) {
		parts := strings.Split(dRoll, `#`)
		dRoll = parts[0]

		buffIds := strings.Split(parts[1], `,`)
		for _, buffId := range buffIds {
			buffId = strings.TrimSpace(buffId)
			buffIdInt, _ := strconv.Atoi(buffId)
			if buffIdInt != 0 {
				buffOnCrit = append(buffOnCrit, buffIdInt)
			}
		}
	}

	invertCount := 1
	if dRoll[0] == '-' {
		dRoll = strings.TrimLeft(dRoll, `-`)
		invertCount = -1
	}

	// 1d3+2, 1d3-1, etc
	// Determine if the bonus is negative or positive
	bonusFactor := 1
	if strings.Contains(dRoll, `+`) {
		dice = strings.Split(dRoll, `+`)
	} else if strings.Contains(dRoll, `-`) {
		bonusFactor = -1 // Invert the bonus
		dice = strings.Split(dRoll, `-`)
	} else {
		dice = []string{dRoll}
	}

	// Apply bonus
	if len(dice) == 2 {
		dice[1] = strings.TrimSpace(dice[1])
		bonus, _ = strconv.Atoi(dice[1])
		bonus *= bonusFactor
	}

	// Parse the dice details
	die := dice[0]

	// How many times does this dice roll get?
	// Only override attacks if we have a valid attack argument provided
	// 2@1d3+2 etc
	attackParts := strings.Split(die, `@`)
	if len(attackParts) == 2 {
		attacks, _ = strconv.Atoi(attackParts[0])
		die = attackParts[1]
	}

	// 2d4 etc.
	dieParts := strings.Split(die, `d`)
	if len(dieParts) == 2 {

		dieParts[0] = strings.TrimSpace(dieParts[0])
		dieParts[1] = strings.TrimSpace(dieParts[1])

		dCount, _ = strconv.Atoi(dieParts[0])
		dSides, _ = strconv.Atoi(dieParts[1])
	}

	return attacks, invertCount * dCount, dSides, bonus, buffOnCrit
}

func FormatDiceRoll(attacks int, dCount int, dSides int, bonus int, buffOnCrit []int) string {

	dRoll := ``

	// 2@
	if attacks != 1 {
		dRoll = fmt.Sprintf(`%d@`, attacks)
	}

	// 2d6
	dRoll += fmt.Sprintf(`%dd%d`, dCount, dSides)

	// +2
	if bonus != 0 {
		if bonus > 0 {
			dRoll += fmt.Sprintf(`+%d`, bonus)
		} else {
			dRoll += fmt.Sprintf(`-%d`, bonus*-1)
		}
	}

	// #9,11,30
	if len(buffOnCrit) > 0 {
		dRoll += `#`
		for _, buffId := range buffOnCrit {
			dRoll = fmt.Sprintf(`%s%d,`, dRoll, buffId)
		}
		dRoll = strings.TrimRight(dRoll, `,`)
	}

	return dRoll
}

// SafeSave first saves to a temp file, then renames it to save over the target destination
// This is to lessen the risk of a partial write being interrupted and corrupting the file
// due to power loss etc.
func SafeSave(path string, data []byte) error {

	path = filepath.FromSlash(path)

	safePath := path + `.new`

	if err := os.WriteFile(safePath, data, 0777); err != nil {
		return err
	}

	//
	// Once the file is written, rename it to remove the .new suffix and overwrite the old file
	//
	if err := os.Rename(safePath, path); err != nil {
		return err
	}

	return nil
}

// Basic save wrapper
func Save(path string, data []byte, doSafe ...bool) error {

	path = filepath.FromSlash(path)

	if len(doSafe) > 0 && doSafe[0] {
		return SafeSave(path, data)
	}

	if err := os.WriteFile(path, data, 0777); err != nil {
		return err
	}

	return nil
}

func FilePath(pathParts ...string) string {
	if len(pathParts) == 1 {
		return filepath.FromSlash(pathParts[0])
	}
	return filepath.FromSlash(strings.Join(pathParts, ``))
}

func BreakIntoParts(full string) []string {
	result := []string{full}

	parts := strings.Split(full, ` `)
	partCt := len(parts)
	for i := 1; i < partCt; i++ {
		result = append(result, strings.Join(parts[i:], ` `))
	}

	return result
}

func HealthClass(health int, maxHealth int) string {

	if health <= 0 {
		return `health-dead`
	}
	// quantize to 10s
	healthPercent := int(math.Floor(float64(health)/float64(maxHealth)*10)) * 10

	return fmt.Sprintf(`health-%d`, healthPercent)
}

// Creates a percentage and quantizes it to the nearest 10
func QuantizeTens(value int, max int) int {
	return int(math.Floor(float64(value)/float64(max)*10)) * 10
}

// Strips out common prepositions from a string
func StripPrepositions(input string) string {

	if input == `` {
		return input
	}

	for _, prep := range strippablePrepositions {
		prepLen := len(prep)

		if len(input) > prepLen && input[0:len(prep)+1] == prep+` ` {
			input = input[len(prep)+1:]
		}
		input = strings.ReplaceAll(input, ` `+prep+` `, ` `)
	}

	return input
}

func ConvertColorShortTags(input string) string {

	colorShortTagRegex = regexp.MustCompile(`\{(\d*)(?::)?(\d*)?\}`)
	if colorShortTagRegex.MatchString(input) {
		input = `<ansi>` + colorShortTagRegex.ReplaceAllString(input, `</ansi><ansi fg="$1" bg="$2">`) + `</ansi>`

		input = strings.ReplaceAll(input, ` bg=""`, ``)
		input = strings.ReplaceAll(input, ` fg=""`, ``)
		input = strings.ReplaceAll(input, `<ansi></ansi>`, ``)
		input = strings.ReplaceAll(input, `</ansi></ansi>`, ``)
	}

	return input
}

func PercentOfTotal(value1 int, value2 int) float64 {
	if value1 == 0 {
		return 0
	}
	return (float64(value1) + float64(value2)) / float64(value1)
}

// Make everything lowercase
// Convert anything that isn't a-z, 0-9 into _
func ConvertForFilename(input string) string {

	s := []byte(strings.ToLower(input))

	pos := 0
	for _, b := range s {
		if b == '\'' { // skip apostrophes
			continue
		} else if ('a' <= b && b <= 'z') || ('0' <= b && b <= '9') {
			s[pos] = b
		} else {
			s[pos] = '_' // If not in the allowed range, convert to underscore
		}
		pos++
	}

	return string(s[0:pos])
}

func StringWildcardMatch(stringToSearch string, patternToSearch string) bool {

	if stringToSearch == patternToSearch {
		return true
	}

	startsWith := false
	endsWith := false

	if patternToSearch[0] == '*' {
		endsWith = true
		patternToSearch = patternToSearch[1:]
	}

	if len(patternToSearch) > 1 && patternToSearch[len(patternToSearch)-1] == '*' {
		startsWith = true
		patternToSearch = patternToSearch[0 : len(patternToSearch)-1]
	}

	if startsWith && !endsWith { // if it starts with
		return strings.HasPrefix(stringToSearch, patternToSearch)
	} else if endsWith && !startsWith { // if it ends with
		return strings.HasSuffix(stringToSearch, patternToSearch)
	} else if startsWith && endsWith {
		return strings.Contains(stringToSearch, patternToSearch)
	}

	return stringToSearch == patternToSearch
}

func ValidateWorldFiles(exampleWorldPath string, worldPath string) error {

	entries, err := os.ReadDir(exampleWorldPath)
	if err != nil {
		return fmt.Errorf("unable to read directory %s: %v", exampleWorldPath, err)
	}

	var subfolders []string
	// Filter out only directories
	for _, entry := range entries {
		if entry.IsDir() {
			subfolders = append(subfolders, entry.Name())
		}
	}

	// Check each source subfolder in the target directory
	for _, folder := range subfolders {
		testPath := filepath.Join(worldPath, folder)

		info, err := os.Stat(testPath)
		if err != nil {
			return fmt.Errorf("'%s' missing folder '%s': %v", worldPath, folder, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("'%s' exists but is not a directory", testPath)
		}
	}

	return nil
}

func BoolYN(b bool) string {
	if b {
		return `yes`
	}
	return `no`
}

func SaveRoundCount(fpath string) {

	err := os.WriteFile(fpath, []byte(strconv.FormatUint(roundCount, 10)), 0644)
	if err != nil {
		mudlog.Error("SaveRoundCount()", "error", err)
	}

}

func LoadRoundCount(fpath string) uint64 {

	roundCountData, err := os.ReadFile(fpath)
	if err != nil {
		roundCount = RoundCountMinimum
		mudlog.Warn("LoadRoundCount()", "error", err, "message", "Trying to create... (First time running?)")
		SaveRoundCount(fpath)
		roundCountData = []byte(strconv.FormatUint(roundCount, 10))
	}

	roundCountUint64, err := strconv.ParseUint(string(roundCountData), 10, 64)
	if err != nil {
		mudlog.Warn("LoadRoundCount()", "error", err, "file-contents", string(roundCountData))
	} else {
		roundCount = roundCountUint64
	}

	if roundCount < RoundCountMinimum {
		roundCount = RoundCountMinimum
	}

	return roundCount
}

func StripANSI(str string) string {
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansi.ReplaceAllString(str, "")
}

func FormatNumber(n int) string {
	in := strconv.Itoa(n)
	numOfDigits := len(in)
	if n < 0 {
		numOfDigits-- // First character is the - sign (not a digit)
	}
	numOfCommas := (numOfDigits - 1) / 3

	out := make([]byte, len(in)+numOfCommas)
	if n < 0 {
		in, out[0] = in[1:], '-'
	}

	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ','
		}
	}
}

func StripCharsForScreenReaders(s string) string {

	// leave [ and ; off this list, it's special for ansi escape codes.
	toReplace := "┌─┐└┘╔═╗╚╝│─•]╒═╕█░╲╱+"

	// Create a lookup map for constant-time rune checks
	replaceSet := make(map[rune]struct{}, len(toReplace))
	for _, r := range toReplace {
		replaceSet[r] = struct{}{}
	}

	strLen := len(s)
	var b strings.Builder
	b.Grow(strLen) // Pre-allocate some memory

	ignoreNext := 0
	for pos, r := range s {

		if ignoreNext > 0 {
			ignoreNext--
			continue
		}

		if r == '[' && (pos == 0 || s[pos-1] != 27) {
			b.WriteRune(' ')
			continue
		}

		if r == '.' && pos < strLen-1 && s[pos+1] == ':' {
			b.WriteRune(' ')
			b.WriteRune(' ')
			ignoreNext = 1
			continue
		}

		if _, found := replaceSet[r]; found {
			b.WriteRune(' ')
		} else {
			b.WriteRune(r)
		}
	}

	return b.String()
}
