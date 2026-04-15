package questengine

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

const (
	LogVerbose = "verbose"
	LogMedium  = "medium"
	LogMinimal = "minimal"
)

// Per-player debug mode tracking
var debugPlayers = make(map[int]bool)

func SetPlayerDebug(userId int, enabled bool) {
	if enabled {
		debugPlayers[userId] = true
	} else {
		delete(debugPlayers, userId)
	}
}

func IsPlayerDebug(userId int) bool {
	return debugPlayers[userId]
}

func getLogLevel() string {
	return string(configs.GetBalanceConfig().QuestLogLevel)
}

func shouldLog(level string, userId int) bool {
	if IsPlayerDebug(userId) {
		return true
	}
	configLevel := getLogLevel()
	switch configLevel {
	case LogVerbose:
		return true
	case LogMedium:
		return level == LogMedium || level == LogMinimal
	case LogMinimal:
		return level == LogMinimal
	}
	return true
}

func LogVerboseF(userId int, format string, args ...any) {
	if shouldLog(LogVerbose, userId) {
		mudlog.Info("[QUEST]", "detail", fmt.Sprintf(format, args...), "userId", userId)
	}
}

func LogMediumF(userId int, format string, args ...any) {
	if shouldLog(LogMedium, userId) {
		mudlog.Info("[QUEST]", "detail", fmt.Sprintf(format, args...), "userId", userId)
	}
}

func LogMinimalF(userId int, format string, args ...any) {
	if shouldLog(LogMinimal, userId) {
		mudlog.Info("[QUEST]", "detail", fmt.Sprintf(format, args...), "userId", userId)
	}
}

func LogError(format string, args ...any) {
	mudlog.Error("[QUEST]", "detail", fmt.Sprintf(format, args...))
}

func LogWarn(format string, args ...any) {
	mudlog.Warn("[QUEST]", "detail", fmt.Sprintf(format, args...))
}

func FormatConditions(c Conditions) string {
	var parts []string
	if len(c.Has) > 0 {
		parts = append(parts, fmt.Sprintf("has:%v", c.Has))
	}
	if len(c.Missing) > 0 {
		parts = append(parts, fmt.Sprintf("missing:%v", c.Missing))
	}
	if c.InRoom > 0 {
		parts = append(parts, fmt.Sprintf("in_room:%d", c.InRoom))
	}
	if c.HasItem > 0 {
		parts = append(parts, fmt.Sprintf("has_item:%d", c.HasItem))
	}
	if c.MissingItem > 0 {
		parts = append(parts, fmt.Sprintf("missing_item:%d", c.MissingItem))
	}
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, ", ")
}
