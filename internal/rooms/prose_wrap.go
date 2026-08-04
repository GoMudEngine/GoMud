package rooms

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

// ////////////////////////////////////////////////////////////////////////////
// PROSE FOLDING FOR SAVED ROOM TEMPLATES
// ////////////////////////////////////////////////////////////////////////////
//
// Authored room YAML wraps long prose with a folded block scalar:
//
//	description: >
//	  A narrow cut climbs north through the cliffs toward the steppe rim,
//	  where a broken silhouette stands black against the sky ...
//
// Loading a folded scalar joins its lines with single spaces, so the value in
// memory is ONE long line. yaml.v2 then re-emits it as a literal block (`|`)
// holding a single enormous line, because the value carries a trailing
// newline. Saving a room through the admin web builder therefore destroyed the
// authored wrapping and made future diffs of that file unreadable.
//
// The fix re-folds those values on the way out. The non-obvious requirement,
// and the thing every decision below is subordinate to, is ROUND-TRIP
// FIDELITY: for any room, load(save(room)).Description must equal
// room.Description exactly. A folded scalar is the right tool precisely
// because loading one re-joins the wrapped lines with single spaces, giving
// back the identical string. A literal (`|`) scalar would NOT do this: it
// would bake hard newlines into the value and change what players see.
//
// Because a mis-folded scalar silently CHANGES content rather than merely
// looking wrong, the code refuses to fold anything it cannot prove safe, and
// verifies the finished document by parsing it back and comparing the prose
// fields. Any doubt at all falls back to plain yaml.v2 output.
//
// ////////////////////////////////////////////////////////////////////////////

// proseWrapWidth is the widest on-disk column a folded prose line may reach.
// The repo convention for player-facing text is 80 columns, so authored files
// wrap a little under that.
const proseWrapWidth = 78

// proseFoldSentinelFormat builds the placeholder written into a working copy of
// the room in place of a value we intend to fold. It is deliberately all
// letters and digits so yaml.v2 always emits it as a short, unquoted, plain
// scalar occupying exactly one line, which is what makes locating it in the
// output reliable.
const proseFoldSentinelFormat = `zzprosefoldzz%dzz`

// marshalRoomTemplate renders a room template to YAML. It is a drop-in
// replacement for yaml.Marshal(&roomTpl) that additionally re-folds long prose
// values, and falls back to exactly that call whenever folding cannot be shown
// to be lossless.
func marshalRoomTemplate(roomTpl Room) ([]byte, error) {

	plain, err := yaml.Marshal(&roomTpl)
	if err != nil {
		return nil, err
	}

	if folded, ok := foldRoomProse(roomTpl, plain); ok {
		return folded, nil
	}

	return plain, nil
}

// foldRoomProse produces a copy of the marshalled room with its long prose
// values rewritten as folded block scalars. It reports false when nothing was
// foldable or when the result failed verification, in which case the caller
// must use the plain output unchanged.
func foldRoomProse(roomTpl Room, plain []byte) ([]byte, bool) {

	// A room parsed back from the untouched yaml.v2 output is the reference
	// the folded document has to match. Comparing against this rather than
	// against roomTpl directly keeps nil-versus-empty container differences
	// from being mistaken for content loss.
	var base Room
	if err := yaml.Unmarshal(plain, &base); err != nil {
		return nil, false
	}

	// Whether a value can be folded depends partly on the indentation yaml.v2
	// chose for it, which is not known until after marshalling. Sentinels that
	// turn out not to be viable are recorded here and the room is marshalled
	// again without them. Dropping one sentinel does not move any other, so a
	// single retry always suffices; the extra rounds are belt and braces.
	skip := map[string]bool{}

	for attempt := 0; attempt < 4; attempt++ {

		work, bodies := collectProseFolds(roomTpl, skip)
		if len(bodies) == 0 {
			return nil, false
		}

		out, err := yaml.Marshal(&work)
		if err != nil {
			return nil, false
		}

		folded, dropped, err := applyProseFolds(out, bodies)
		if err != nil {
			return nil, false
		}

		if len(dropped) > 0 {
			for _, sentinel := range dropped {
				skip[sentinel] = true
			}
			continue
		}

		if !proseRoundTrips(folded, base) {
			return nil, false
		}

		return folded, true
	}

	return nil, false
}

// collectProseFolds returns a copy of roomTpl with every foldable prose value
// replaced by a sentinel, alongside a sentinel to original-value map. Sentinel
// numbering is stable across calls for the same room, so a caller may add a
// sentinel to skip and collect again to get the same assignment minus that one.
//
// The copy never shares the maps or slice it rewrites, so the caller's room is
// left untouched.
func collectProseFolds(roomTpl Room, skip map[string]bool) (Room, map[string]string) {

	work := roomTpl
	bodies := map[string]string{}

	next := 0
	claim := func(body string) (string, bool) {
		sentinel := fmt.Sprintf(proseFoldSentinelFormat, next)
		next++
		if skip[sentinel] || !canFoldProse(body) {
			return ``, false
		}
		bodies[sentinel] = body
		return sentinel, true
	}

	if sentinel, ok := claim(roomTpl.Description); ok {
		work.Description = sentinel
	}

	if len(roomTpl.Nouns) > 0 {
		nouns := make(map[string]string, len(roomTpl.Nouns))
		for k, v := range roomTpl.Nouns {
			nouns[k] = v
		}
		for _, k := range sortedProseKeys(roomTpl.Nouns) {
			if sentinel, ok := claim(roomTpl.Nouns[k]); ok {
				nouns[k] = sentinel
			}
		}
		work.Nouns = nouns
	}

	if len(roomTpl.HiddenNouns) > 0 {
		hidden := make(map[string]HiddenNoun, len(roomTpl.HiddenNouns))
		for k, v := range roomTpl.HiddenNouns {
			hidden[k] = v
		}
		for _, k := range sortedProseKeys(roomTpl.HiddenNouns) {
			entry := hidden[k]
			if sentinel, ok := claim(roomTpl.HiddenNouns[k].Description); ok {
				entry.Description = sentinel
			}
			if sentinel, ok := claim(roomTpl.HiddenNouns[k].HiddenDescription); ok {
				entry.HiddenDescription = sentinel
			}
			hidden[k] = entry
		}
		work.HiddenNouns = hidden
	}

	if len(roomTpl.IdleMessages) > 0 {
		msgs := make([]string, len(roomTpl.IdleMessages))
		copy(msgs, roomTpl.IdleMessages)
		for i, m := range roomTpl.IdleMessages {
			if sentinel, ok := claim(m); ok {
				msgs[i] = sentinel
			}
		}
		work.IdleMessages = msgs
	}

	return work, bodies
}

// canFoldProse reports whether a value's CONTENT permits folding. It says
// nothing about width, which depends on indentation and is checked later.
//
// Every rejection here is a case where a folded scalar would either fail to
// reproduce the value or would need chomping and indentation indicators that
// are not worth the risk on authored content.
func canFoldProse(body string) bool {

	if body == `` {
		return false
	}

	// Clip chomping preserves exactly one trailing newline, which is what a
	// value loaded from an authored `>` block carries. Anything else is out.
	trimmed := strings.TrimSuffix(body, "\n")
	if trimmed == `` {
		return false
	}

	// Interior newlines (and any second trailing one) would need blank lines
	// or a literal block to survive. Carriage returns and tabs fold
	// unpredictably.
	if strings.ContainsAny(trimmed, "\n\r\t") {
		return false
	}

	// A folded line that begins with whitespace is treated literally rather
	// than folded, which changes the value. Runs of spaces are where that
	// would happen, so refuse them outright rather than reason about where a
	// break may land.
	if strings.Contains(trimmed, `  `) {
		return false
	}

	if strings.HasPrefix(trimmed, ` `) || strings.HasSuffix(trimmed, ` `) {
		return false
	}

	return true
}

// applyProseFolds rewrites each sentinel in out into a folded block scalar. It
// returns the rewritten document plus the sentinels that had to be abandoned
// because the emitted line shape or width made a faithful fold impossible; the
// caller re-marshals without those. An error means the output did not look the
// way it must for text rewriting to be safe, and the whole attempt is void.
func applyProseFolds(out []byte, bodies map[string]string) ([]byte, []string, error) {

	lines := strings.Split(string(out), "\n")
	text := string(out)

	replacements := map[int][]string{}
	dropped := []string{}

	for _, sentinel := range sortedProseKeys(bodies) {

		// The sentinel stands in for a whole value, so it must appear exactly
		// once. Anything else means content collided with it and rewriting
		// would hit the wrong place.
		if n := strings.Count(text, sentinel); n != 1 {
			return nil, nil, fmt.Errorf(`prose fold sentinel %s appeared %d times`, sentinel, n)
		}

		idx := -1
		for i, line := range lines {
			if strings.Contains(line, sentinel) {
				idx = i
				break
			}
		}
		if idx < 0 {
			return nil, nil, fmt.Errorf(`prose fold sentinel %s spans lines`, sentinel)
		}

		line := lines[idx]

		// The sentinel must be the entire value, unquoted, at the end of the
		// line. If yaml.v2 quoted or wrapped it, leave the value alone.
		if !strings.HasSuffix(line, sentinel) {
			dropped = append(dropped, sentinel)
			continue
		}

		prefix := line[:len(line)-len(sentinel)]
		if !strings.HasSuffix(prefix, `: `) && !strings.HasSuffix(prefix, `- `) {
			dropped = append(dropped, sentinel)
			continue
		}

		indent := len(prefix) - len(strings.TrimLeft(prefix, ` `))
		contentIndent := indent + 2

		body := bodies[sentinel]
		trimmed := strings.TrimSuffix(body, "\n")

		// Already inside the width on the line yaml.v2 gave it. Folding would
		// be churn for nothing, so hand it back to the plain emitter.
		if len(prefix)+len(trimmed) <= proseWrapWidth {
			dropped = append(dropped, sentinel)
			continue
		}

		wrapped, ok := wrapProse(trimmed, proseWrapWidth-contentIndent)
		if !ok {
			dropped = append(dropped, sentinel)
			continue
		}

		// Clip (`>`) keeps the single trailing newline the value carries.
		// Strip (`>-`) is for values that have none. Getting this backwards
		// adds or removes a newline, so it is derived from the value itself.
		header := `>-`
		if strings.HasSuffix(body, "\n") {
			header = `>`
		}

		block := make([]string, 0, len(wrapped)+1)
		block = append(block, prefix+header)
		pad := strings.Repeat(` `, contentIndent)
		for _, w := range wrapped {
			block = append(block, pad+w)
		}

		replacements[idx] = block
	}

	if len(dropped) > 0 {
		return nil, dropped, nil
	}

	final := make([]string, 0, len(lines)+len(bodies)*8)
	for i, line := range lines {
		if block, ok := replacements[i]; ok {
			final = append(final, block...)
			continue
		}
		final = append(final, line)
	}

	return []byte(strings.Join(final, "\n")), nil, nil
}

// wrapProse greedily breaks body into lines of at most width characters,
// breaking only at single spaces. It reports false when the break points
// cannot be chosen without altering the value, which is the case when a single
// token is wider than the line.
//
// The final guard is the important one: the lines must rejoin with single
// spaces back into exactly the input, which is precisely what YAML folding
// will do when the value is read again.
func wrapProse(body string, width int) ([]string, bool) {

	if width < 20 || body == `` {
		return nil, false
	}

	words := strings.Split(body, ` `)

	lines := []string{}
	cur := ``

	for _, w := range words {
		if w == `` {
			// A run of spaces. canFoldProse should have caught this.
			return nil, false
		}
		if len(w) > width {
			return nil, false
		}
		if cur == `` {
			cur = w
			continue
		}
		if len(cur)+1+len(w) <= width {
			cur += ` ` + w
			continue
		}
		lines = append(lines, cur)
		cur = w
	}

	if cur != `` {
		lines = append(lines, cur)
	}

	if len(lines) < 2 {
		return nil, false
	}

	if strings.Join(lines, ` `) != body {
		return nil, false
	}

	return lines, true
}

// proseRoundTrips parses the folded document back and confirms every prose
// field still holds exactly what the untouched yaml.v2 output produced. This
// is the guarantee the whole file exists to provide, so it is checked on the
// finished bytes rather than inferred from the steps that made them.
func proseRoundTrips(folded []byte, base Room) bool {

	var check Room
	if err := yaml.Unmarshal(folded, &check); err != nil {
		return false
	}

	if check.Description != base.Description {
		return false
	}
	if !reflect.DeepEqual(check.Nouns, base.Nouns) {
		return false
	}
	if !reflect.DeepEqual(check.HiddenNouns, base.HiddenNouns) {
		return false
	}
	if !reflect.DeepEqual(check.IdleMessages, base.IdleMessages) {
		return false
	}

	return true
}

func sortedProseKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
