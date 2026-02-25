package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// AskAsync fires a goroutine that calls Ollama and delivers the response via onResponse.
// onResponse is called under util.LockMud() — safe to call mob/room functions directly.
// onUnavailable is called (without the mud lock) when the LLM is down, timed out, or busy.
func AskAsync(
	profile *LLMProfile,
	endpoint string,
	timeoutSecs int,
	mobInstanceId int,
	ctx ConversationContext,
	topic string,
	onResponse func(response string),
	onUnavailable func(),
) {
	go func() {
		// 1. If another request is already in-flight for this mob, fall back immediately.
		if isPending(mobInstanceId) {
			mudlog.Info(`llm`, `status`, fmt.Sprintf("mob %d already pending, falling back to YAML", mobInstanceId))
			onUnavailable()
			return
		}

		// 2. Check response cache.
		if cached, ok := checkCache(mobInstanceId, topic); ok {
			mudlog.Info(`llm`, `status`, fmt.Sprintf("mob %d cache hit for topic %q", mobInstanceId, topic))
			util.LockMud()
			onResponse(cached)
			util.UnlockMud()
			return
		}

		// 3. Mark as in-flight.
		setPending(mobInstanceId, true)
		defer setPending(mobInstanceId, false)

		// 4. Build system prompt.
		maxWords := profile.MaxWords
		if maxWords <= 0 {
			maxWords = 80
		}
		recentStr := "none"
		if len(ctx.RecentTopics) > 0 {
			recentStr = strings.Join(ctx.RecentTopics, ", ")
		}
		questStr := ""
		if len(ctx.QuestContext) > 0 {
			questStr = fmt.Sprintf("\nPlayer's active quests relevant to you: %s.", strings.Join(ctx.QuestContext, "; "))
		}
		systemPrompt := fmt.Sprintf(
			"%s\nYou are %s in %s. Keep responses under %d words. Stay in character.\nCurrent mood: %s.\nRecent topics discussed: %s.%s",
			profile.SystemPrompt,
			ctx.MobName, ctx.ZoneName, maxWords,
			ctx.CurrentMood,
			recentStr,
			questStr,
		)

		reqBody := ollamaRequest{
			Model: profile.Model,
			Messages: []ollamaMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: topic},
			},
			Stream: false,
		}
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			mudlog.Error(`llm`, `error`, fmt.Sprintf("failed to marshal request: %v", err))
			onUnavailable()
			return
		}

		// 5. Fire the HTTP request with a timeout.
		if timeoutSecs <= 0 {
			timeoutSecs = 10
		}
		httpCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSecs)*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, endpoint+"/api/chat", bytes.NewReader(bodyBytes))
		if err != nil {
			mudlog.Error(`llm`, `error`, fmt.Sprintf("failed to build request: %v", err))
			onUnavailable()
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			mudlog.Info(`llm`, `status`, fmt.Sprintf("ollama unavailable for mob %d: %v", mobInstanceId, err))
			onUnavailable()
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			mudlog.Info(`llm`, `status`, fmt.Sprintf("ollama returned status %d for mob %d", resp.StatusCode, mobInstanceId))
			onUnavailable()
			return
		}

		// 6. Decode response.
		var ollamaResp ollamaResponse
		if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
			mudlog.Error(`llm`, `error`, fmt.Sprintf("failed to decode ollama response: %v", err))
			onUnavailable()
			return
		}

		response := strings.TrimSpace(ollamaResp.Message.Content)
		if response == "" {
			mudlog.Info(`llm`, `status`, fmt.Sprintf("empty response from ollama for mob %d", mobInstanceId))
			onUnavailable()
			return
		}

		// 7. Cache and deliver under mud lock.
		ttl := profile.CacheTTL
		if ttl == "" {
			ttl = "1h"
		}
		storeCache(mobInstanceId, topic, response, ttl)

		mudlog.Info(`llm`, `status`, fmt.Sprintf("mob %d LLM response delivered for topic %q", mobInstanceId, topic))
		util.LockMud()
		onResponse(response)
		util.UnlockMud()
	}()
}
