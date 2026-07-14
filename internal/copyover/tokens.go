package copyover

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const tokenTTL = 2 * time.Minute

type tokenEntry struct {
	UserId    int       `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type tokenState struct {
	Tokens map[string]tokenEntry `json:"tokens"`
}

var (
	tokenMu    sync.Mutex
	tokenStore = map[string]tokenEntry{}
)

// IssueReconnectToken creates a one-time reconnect token for the given userId
// and returns it. The token expires after 2 minutes.
func IssueReconnectToken(userId int) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)

	tokenMu.Lock()
	tokenStore[token] = tokenEntry{
		UserId:    userId,
		ExpiresAt: time.Now().Add(tokenTTL),
	}
	tokenMu.Unlock()

	return token, nil
}

// ConsumeReconnectToken validates and consumes a token, returning the
// associated userId. Returns 0, false if the token is unknown or expired.
func ConsumeReconnectToken(token string) (int, bool) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	entry, ok := tokenStore[token]
	if !ok {
		return 0, false
	}

	delete(tokenStore, token)

	if time.Now().After(entry.ExpiresAt) {
		return 0, false
	}

	return entry.UserId, true
}

func pruneExpiredTokens() {
	now := time.Now()
	for tok, entry := range tokenStore {
		if now.After(entry.ExpiresAt) {
			delete(tokenStore, tok)
		}
	}
}

type tokenContributor struct{}

func (t *tokenContributor) CopyoverName() string { return "reconnect_tokens" }

func (t *tokenContributor) CopyoverSave(enc *Encoder) error {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	pruneExpiredTokens()

	state := tokenState{Tokens: make(map[string]tokenEntry, len(tokenStore))}
	for k, v := range tokenStore {
		state.Tokens[k] = v
	}

	return enc.WriteSection(t.CopyoverName(), state)
}

func (t *tokenContributor) CopyoverRestore(dec *Decoder) error {
	var state tokenState
	if err := dec.ReadSection(t.CopyoverName(), &state); err != nil {
		return err
	}

	tokenMu.Lock()
	defer tokenMu.Unlock()

	tokenStore = state.Tokens
	if tokenStore == nil {
		tokenStore = map[string]tokenEntry{}
	}

	return nil
}

// TokenContributor returns the Contributor that persists reconnect tokens
// across a copyover. Register it in main alongside the other contributors.
func TokenContributor() Contributor {
	return &tokenContributor{}
}
