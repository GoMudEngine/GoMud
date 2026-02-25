package discord

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/ansitags"
)

var (
	lastMonth = -1
)

// Player enters the world event
func HandlePlayerSpawn(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		return events.Cancel
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Cancel
	}

	connDetails := connections.Get(user.ConnectionId())

	message := fmt.Sprintf(":white_check_mark: **%s** connected", user.Character.Name)

	if connDetails.IsWebSocket() {
		message += ` (via websocket)`
	}

	SendRichMessage(message, Green)

	return events.Continue
}

// Player leaves the world event
func HandlePlayerDespawn(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PlayerDespawn)
	if !typeOk {
		return events.Cancel
	}

	message := fmt.Sprintf(":x: **%s** disconnected (online %s)", evt.CharacterName, evt.TimeOnline)

	SendRichMessage(message, Grey)

	return events.Continue
}

func HandleLogs(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.Log)
	if !typeOk {
		return events.Cancel
	}

	if evt.Level != `ERROR` {
		return events.Continue
	}

	msgOut := util.StripANSI(fmt.Sprintln(evt.Data[1:]...))

	// Skip script timeout messages
	if strings.Contains(msgOut, `JSVM`) && strings.Contains(msgOut, `script timeout`) {
		return events.Continue
	}

	if strings.Contains(msgOut, `Stopping server`) || strings.Contains(msgOut, `Starting server`) {
		msgOut = strings.Replace(msgOut, evt.Level, `**NOTICE**`, 1)
	} else {
		msgOut = strings.Replace(msgOut, evt.Level, `**`+evt.Level+`**`, 1)
	}

	message := fmt.Sprintf(":bangbang: %s", msgOut)

	SendRichMessage(message, Red)

	return events.Continue
}

func HandleDeath(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.PlayerDeath)
	if !typeOk {
		return events.Cancel
	}

	message := fmt.Sprintf(`:skull: **%s** *has **DIED**!*`, evt.CharacterName)
	SendRichMessage(message, DarkOrange)

	return events.Continue
}

func HandleBroadcast(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.Broadcast)
	if !typeOk {
		return events.Cancel
	}

	if !evt.IsCommunication {
		return events.Continue
	}

	textOut := ansitags.Parse(evt.Text, ansitags.StripTags)

	textOut = strings.ReplaceAll(textOut, "\n", "")
	textOut = strings.ReplaceAll(textOut, "\r", "")
	textOut = strings.Replace(textOut, `(broadcast) `, `(broadcast) **`, 1)
	textOut = strings.Replace(textOut, `:`, `:** `, 1)

	message := fmt.Sprintf(`:speech_balloon: %s`, textOut)

	SendRichMessage(message, Purple)

	return events.Continue
}

func HandleAuctionUpdate(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.GenericEvent)
	if !typeOk {
		return events.Cancel
	}

	// Extract event details
	var ok bool

	var EventState string
	if EventState, ok = evt.Data(`State`).(string); !ok {
		EventState = ``
	}

	var EventItemName string
	if EventItemName, ok = evt.Data(`ItemName`).(string); !ok {
		EventItemName = `Unknown`
	}

	var EventItemDescription string
	if EventItemDescription, ok = evt.Data(`ItemDescription`).(string); !ok {
		EventItemDescription = `Unknown`
	}

	var EventSellerName string
	if EventSellerName, ok = evt.Data(`SellerName`).(string); !ok {
		EventSellerName = `Unknown`
	}

	var EventBuyerName string
	if EventBuyerName, ok = evt.Data(`BuyerName`).(string); !ok {
		EventBuyerName = `Unknown`
	}

	var EventBidAmount int
	if EventBidAmount, ok = evt.Data(`BidAmount`).(int); !ok {
		EventBidAmount = 0
	}

	// Process event

	// Don't spam the reminders.
	if EventState == `REMINDER` {
		return events.Continue
	}

	if EventState == `BID` {

		itemName := ansitags.Parse(EventItemName, ansitags.StripTags)

		payload := webHookPayload{
			Embeds: []embed{{
				Color:       Gold,
				Description: fmt.Sprintf(`:moneybag: **%s** has bid on the auction!`, EventBuyerName),
				Fields: []embedField{
					{
						Name:   `Amount`,
						Value:  strconv.Itoa(EventBidAmount),
						Inline: false,
					},
					{
						Name:   `Item`,
						Value:  itemName,
						Inline: true,
					},
					{
						Name:   `Description`,
						Value:  EventItemDescription,
						Inline: true,
					},
				},
			}},
		}

		SendPayload(payload)

		return events.Continue
	}

	if EventState == `START` {

		itemName := ansitags.Parse(EventItemName, ansitags.StripTags)

		payload := webHookPayload{
			Embeds: []embed{{
				Color:       Gold,
				Description: fmt.Sprintf(`:moneybag: **%s** has started a new auction!`, EventSellerName),
				Fields: []embedField{
					{
						Name:   `Item`,
						Value:  itemName,
						Inline: true,
					},
					{
						Name:   `Description`,
						Value:  EventItemDescription,
						Inline: true,
					},
				},
			}},
		}

		SendPayload(payload)

		return events.Continue
	}

	if EventState == `END` {

		itemName := ansitags.Parse(EventItemName, ansitags.StripTags)

		auctionWinner := `No Winner`
		highestBid := `No Bids`

		if EventBidAmount > 0 {
			auctionWinner = EventBuyerName
			highestBid = strconv.Itoa(EventBidAmount)
		}

		payload := webHookPayload{
			Embeds: []embed{{
				Color:       Gold,
				Description: `:moneybag: The auction has ended!`,
				Fields: []embedField{
					{
						Name:   `Highest Bid`,
						Value:  highestBid,
						Inline: true,
					},
					{
						Name:   `Winner`,
						Value:  auctionWinner,
						Inline: true,
					},
					{
						Name:   ` `,
						Value:  ` `,
						Inline: false,
					},
					{
						Name:   `Item`,
						Value:  itemName,
						Inline: true,
					},
					{
						Name:   `Description`,
						Value:  EventItemDescription,
						Inline: true,
					},
				},
			}},
		}

		SendPayload(payload)

		return events.Continue
	}

	return events.Continue
}
