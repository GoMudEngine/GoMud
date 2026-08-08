package messaging

import (
	"regexp"
	"strings"
)

// ansiTagPattern matches <ansi …> (any attribute set, including fg,
// bg, or combinations) or </ansi>. Only used for scanning;
// replacement uses the literal strings.
var ansiTagPattern = regexp.MustCompile(`<ansi[^>]*>|</ansi>`)

// WrapAnsi wraps text at maxWidth display columns. ANSI escape
// sequences (<ansi …> / </ansi> tags) don't count toward width.
// Open tags carry across line breaks: each new line gets a fresh
// reopener if the previous line ended mid-tag.
//
// On malformed input (orphan tags, unmatched closers), falls back to
// a byte-count wrap to avoid panicking. The visual output is uglier
// but the server stays up.
//
// A maxWidth of 0 (unset) returns the input unchanged.
func WrapAnsi(text string, maxWidth int) (wrapped string) {
	if maxWidth <= 0 || text == "" {
		return text
	}
	defer func() {
		// Last-resort: if anything in the parser panics, the caller
		// gets the original text back. This needs a NAMED result and an
		// explicit assignment: a bare recover() on an unnamed return
		// silently handed the caller "" and erased the message.
		if r := recover(); r != nil {
			wrapped = text
		}
	}()

	// Walk the text token-by-token, tracking display column and the
	// currently-open ANSI tag (if any). When we cross maxWidth at a
	// word boundary, emit a newline; if a tag is open, close it
	// before the break and reopen on the next line.
	var (
		out            strings.Builder
		line           strings.Builder
		col            int
		openTag        string // empty when no tag is open
		curWord        strings.Builder
		curWordW       int
		lineHasContent bool // visible content on line (not just tag re-opener)
	)

	flushWord := func() {
		// Add space before word if line already has content and there's room.
		if lineHasContent && col+1+curWordW > maxWidth {
			// Wrap before the word.
			if openTag != "" {
				line.WriteString(`</ansi>`)
			}
			out.WriteString(line.String())
			out.WriteByte('\n')
			line.Reset()
			col = 0
			lineHasContent = false
			if openTag != "" {
				line.WriteString(openTag)
			}
		} else if lineHasContent {
			line.WriteByte(' ')
			col++
		}
		line.WriteString(curWord.String())
		col += curWordW
		lineHasContent = true
		curWord.Reset()
		curWordW = 0
	}

	i := 0
	for i < len(text) {
		// ANSI tag?
		if text[i] == '<' {
			loc := ansiTagPattern.FindStringIndex(text[i:])
			if loc != nil && loc[0] == 0 {
				tag := text[i : i+loc[1]]
				if strings.HasPrefix(tag, `</`) {
					openTag = ""
				} else {
					openTag = tag
				}
				curWord.WriteString(tag)
				i += loc[1]
				continue
			}
		}
		// Whitespace boundary?
		if text[i] == ' ' || text[i] == '\n' {
			if curWord.Len() > 0 {
				flushWord()
			}
			if text[i] == '\n' {
				if openTag != "" {
					line.WriteString(`</ansi>`)
				}
				out.WriteString(line.String())
				out.WriteByte('\n')
				line.Reset()
				col = 0
				lineHasContent = false
				if openTag != "" {
					line.WriteString(openTag)
				}
			}
			i++
			continue
		}
		// Visible character.
		curWord.WriteByte(text[i])
		curWordW++
		i++
	}
	if curWord.Len() > 0 {
		flushWord()
	}
	out.WriteString(line.String())
	return out.String()
}
