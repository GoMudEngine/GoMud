package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/plugins"
)

func init() {
	g := GMCPMutationModule{
		plug: plugins.New(`gmcp.Mutation`, `1.0`),
	}
	events.RegisterListener(mutations.Gained{}, g.onMutationGained)
}

type GMCPMutationModule struct {
	plug *plugins.Plugin
}

// GMCPMutationModule_Payload is pushed as "Char.Mutation" when a player
// acquires (isNew) or deepens a mutation. The web client shows a corner
// toast that expands to the ceremonial card. No mechanical numbers beyond
// rank (used only for "deepens" wording client-side).
type GMCPMutationModule_Payload struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Rank        int    `json:"rank"`
	IsNew       bool   `json:"isNew"`
	Description string `json:"description"`
	Art         string `json:"art"`
}

func buildMutationPayload(evt mutations.Gained) (GMCPMutationModule_Payload, bool) {
	spec := mutations.GetMutation(evt.MutationId)
	if spec == nil {
		return GMCPMutationModule_Payload{}, false
	}
	return GMCPMutationModule_Payload{
		Id:          evt.MutationId,
		Name:        spec.Name,
		Rank:        evt.Rank,
		IsNew:       evt.IsNew,
		Description: spec.Description,
		Art:         `/static/images/mutations/` + evt.MutationId + `.png`,
	}, true
}

func (g *GMCPMutationModule) onMutationGained(e events.Event) events.ListenerReturn {
	evt, ok := e.(mutations.Gained)
	if !ok {
		return events.Continue
	}
	payload, ok := buildMutationPayload(evt)
	if !ok {
		return events.Continue
	}
	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  `Char.Mutation`,
		Payload: payload,
	})
	return events.Continue
}
