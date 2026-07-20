// Reads the reviewed vendor_tag_proposal_reviewed CSV and applies
// vendor_categories tags to item YAMLs via line-based text insertion.
// This preserves existing formatting and comments (no YAML marshalling).
//
// Run:
//
//	go run tools/economy/apply_vendor_tags/main.go
//
// Input: tools/economy/generate_vendor_tags/vendor_tag_proposal_reviewed - vendor_tag_proposal_reviewed.csv
// Modified: _datafiles/world/dogmud/items/**/*.yaml
package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	csvPath  = "tools/economy/generate_vendor_tags/vendor_tag_proposal_reviewed - vendor_tag_proposal_reviewed.csv"
	itemsDir = "_datafiles/world/dogmud/items"
)

func main() {
	proposals, err := readProposals(csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read proposals: %v\n", err)
		os.Exit(1)
	}

	yamlByItemId, err := indexItemYAMLs(itemsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "index items: %v\n", err)
		os.Exit(1)
	}

	applied, skipped, missing := 0, 0, 0
	for itemId, tags := range proposals {
		if len(tags) == 0 {
			skipped++
			continue
		}
		path, ok := yamlByItemId[itemId]
		if !ok {
			fmt.Fprintf(os.Stderr, "WARN: no YAML for itemId %d\n", itemId)
			missing++
			continue
		}
		if err := applyTagsToYAML(path, tags); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s: %v\n", path, err)
			continue
		}
		applied++
	}
	fmt.Printf("Applied vendor_categories to %d files. Skipped %d empty/None rows. Missing %d.\n", applied, skipped, missing)
}

// readProposals returns map[itemId][]tag (sorted, dedup'd).
// Skips rows where proposed_tags is empty, "None", or any variation.
func readProposals(path string) (map[int][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	proposals := map[int][]string{}

	// Skip header
	if len(rows) > 0 {
		rows = rows[1:]
	}

	for _, row := range rows {
		if len(row) < 9 {
			continue
		}

		itemIdStr := strings.TrimSpace(row[0])
		proposedStr := strings.TrimSpace(row[8])

		itemId, err := strconv.Atoi(itemIdStr)
		if err != nil {
			continue
		}

		// Skip empty or "None"
		if proposedStr == "" || strings.EqualFold(proposedStr, "none") {
			continue
		}

		// Parse comma-separated tags
		parts := strings.Split(proposedStr, ",")
		var tags []string
		for _, p := range parts {
			tag := strings.TrimSpace(p)
			if tag != "" {
				tags = append(tags, tag)
			}
		}

		// Dedup and sort
		tagSet := map[string]bool{}
		for _, t := range tags {
			tagSet[t] = true
		}
		tags = nil
		for t := range tagSet {
			tags = append(tags, t)
		}
		sort.Strings(tags)

		if len(tags) > 0 {
			proposals[itemId] = tags
		}
	}

	return proposals, nil
}

// indexItemYAMLs returns map[itemId] -> file path by walking itemsDir.
// Parses itemid: field from each YAML file.
func indexItemYAMLs(dir string) (map[int]string, error) {
	index := map[int]string{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Simple regex to extract itemid: value
		re := regexp.MustCompile(`^\s*itemid:\s*(\d+)`)
		scanner := bufio.NewScanner(strings.NewReader(string(raw)))
		for scanner.Scan() {
			line := scanner.Text()
			matches := re.FindStringSubmatch(line)
			if len(matches) == 2 {
				itemId, err := strconv.Atoi(matches[1])
				if err == nil && itemId > 0 {
					index[itemId] = path
				}
				return nil
			}
		}
		return nil
	})

	return index, err
}

// applyTagsToYAML reads path as text, replaces or inserts vendor_categories block.
// If existing vendor_categories block is present, replace it entirely.
// If absent, insert after is_component: if present, else component_tag:, else name:.
// Format:
//
//	vendor_categories:
//	- tag1
//	- tag2
//	...
func applyTagsToYAML(path string, tags []string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(raw), "\n")

	// Find existing vendor_categories block (line start + following "- " lines)
	vcStartIdx := -1
	vcEndIdx := -1
	for i, line := range lines {
		if matched, _ := regexp.MatchString(`^\s*vendor_categories:\s*$`, line); matched {
			vcStartIdx = i
			// Find end: next non-comment, non-continuation line
			for j := i + 1; j < len(lines); j++ {
				if matched, _ := regexp.MatchString(`^\s*-\s+`, lines[j]); !matched && strings.TrimSpace(lines[j]) != "" {
					vcEndIdx = j - 1
					break
				}
				if j == len(lines)-1 {
					vcEndIdx = j
					break
				}
			}
			break
		}
	}

	// Build replacement block
	var replacement []string
	replacement = append(replacement, "vendor_categories:")
	for _, tag := range tags {
		replacement = append(replacement, "- "+tag)
	}

	var newLines []string
	if vcStartIdx >= 0 {
		// Replace existing block
		newLines = append(newLines, lines[:vcStartIdx]...)
		newLines = append(newLines, replacement...)
		if vcEndIdx+1 < len(lines) {
			newLines = append(newLines, lines[vcEndIdx+1:]...)
		}
	} else {
		// Insert after is_component: or component_tag: or name:
		insertIdx := -1
		for i, line := range lines {
			if matched, _ := regexp.MatchString(`^\s*is_component:`, line); matched {
				insertIdx = i + 1
				break
			}
		}
		if insertIdx < 0 {
			for i, line := range lines {
				if matched, _ := regexp.MatchString(`^\s*component_tag:`, line); matched {
					insertIdx = i + 1
					break
				}
			}
		}
		if insertIdx < 0 {
			for i, line := range lines {
				if matched, _ := regexp.MatchString(`^\s*name:`, line); matched {
					insertIdx = i + 1
					break
				}
			}
		}

		if insertIdx > 0 {
			newLines = append(newLines, lines[:insertIdx]...)
			newLines = append(newLines, replacement...)
			newLines = append(newLines, lines[insertIdx:]...)
		} else {
			// Fallback: append at end
			newLines = append(newLines, lines...)
			newLines = append(newLines, replacement...)
		}
	}

	// Write back, preserving final newline if it existed
	result := strings.Join(newLines, "\n")
	if len(raw) > 0 && raw[len(raw)-1] == '\n' && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return os.WriteFile(path, []byte(result), 0644)
}
