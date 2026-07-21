package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/splash"
)

// GMCPSplashModule delivers splash scenes to web clients as a GMCP Event.Splash
// package (the web client renders the SVG scene inline). Non-web recipients are
// handled by the internal/hooks terminal delivery, so each recipient is served
// exactly once.
type GMCPSplashModule struct {
	plug *plugins.Plugin
}

// SplashPayload is the GMCP Event.Splash body. Art is client-side, keyed by Scene.
type SplashPayload struct {
	Scene   string `json:"scene"`
	Caption string `json:"caption"`
	Data    any    `json:"data,omitempty"`
}

func init() {
	g := GMCPSplashModule{
		plug: plugins.New(`gmcp.Splash`, `1.0`),
	}
	events.RegisterListener(splash.Splash{}, g.deliverSplashToWeb)
}

func (g *GMCPSplashModule) deliverSplashToWeb(e events.Event) events.ListenerReturn {
	evt, ok := e.(splash.Splash)
	if !ok {
		return events.Continue
	}
	if !bool(configs.GetGamePlayConfig().SplashesEnabled) {
		return events.Continue
	}
	payload := SplashPayload{Scene: evt.SceneId, Caption: evt.Caption, Data: evt.Data}
	for _, u := range splash.Recipients(evt) {
		if !connections.GetClientSettings(u.ConnectionId()).IsWebConnection {
			continue // non-web recipients handled by internal/hooks
		}
		events.AddToQueue(GMCPOut{UserId: u.UserId, Module: "Event.Splash", Payload: payload})
	}
	return events.Continue
}
