package auctions

import (
	"embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (

	//////////////////////////////////////////////////////////////////////
	// NOTE: The below //go:embed directive is important!
	// It embeds the relative path into the var below it.
	//////////////////////////////////////////////////////////////////////

	//go:embed files/*
	files embed.FS
)

// ////////////////////////////////////////////////////////////////////
// NOTE: The init function in Go is a special function that is
// automatically executed before the main function within a package.
// It is used to initialize variables, set up configurations, or
// perform any other setup tasks that need to be done before the
// program starts running.
// ////////////////////////////////////////////////////////////////////
func init() {

	//
	// We can use all functions only, but this demonstrates
	// how to use a struct
	//
	a := AuctionsModule{
		plug: plugins.New(`auctions`, `1.0`),
		auctionMgr: AuctionManager{
			ActiveAuction:   nil,
			maxHistoryItems: 10,
			PastAuctions:    []PastAuctionItem{},
		},
	}

	//
	// Add the embedded filesystem
	//
	if err := a.plug.AttachFileSystem(files); err != nil {
		panic(err)
	}
	//
	// Register any user/mob commands
	//
	a.plug.AddUserCommand(`auction`, a.auctionCommand, true, false)
	//
	// Register callbacks for load/unload
	//
	a.plug.Callbacks.SetOnLoad(a.load)
	a.plug.Callbacks.SetOnSave(a.save)

	events.RegisterListener(events.NewRound{}, a.newRoundHandler)
}

//////////////////////////////////////////////////////////////////////
// NOTE: What follows is all custom code. For this module.
//////////////////////////////////////////////////////////////////////

// Using a struct gives a way to store longer term data.
type AuctionsModule struct {
	// Keep a reference to the plugin when we create it so that we can call ReadBytes() and WriteBytes() on it.
	plug *plugins.Plugin

	auctionMgr AuctionManager
}

type AuctionUpdate struct {
	State           string // START, REMINDER, BID, END
	ItemName        string
	ItemDescription string
	SellerName      string
	BuyerName       string
	BidAmount       int
}

func (ae AuctionUpdate) Type() string { return `AuctionUpdate` }
func (ae AuctionUpdate) Data(name string) any {
	switch strings.ToLower(name) {
	case `state`:
		return ae.State
	case `itemname`:
		return ae.ItemName
	case `itemdescription`:
		return ae.ItemDescription
	case `sellername`:
		return ae.SellerName
	case `buyername`:
		return ae.BuyerName
	case `bidamount`:
		return ae.BidAmount
	}
	return nil
}

func (mod *AuctionsModule) load() {
	mod.plug.ReadIntoStruct(`auctionhistory`, &mod.auctionMgr)
}

func (mod *AuctionsModule) save() {
	mod.plug.WriteStruct(`auctionhistory`, mod.auctionMgr)
}

// Module functions
func (mod *AuctionsModule) auctionCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if on := user.GetConfigOption(`auction`); on != nil && !on.(bool) {

		user.SendTextLegacy(
			`Auctions are disabled. See <ansi fg="command">help set</ansi> for learn how to change this.`,
		)

		return true, nil
	}

	currentAuction := mod.auctionMgr.GetCurrentAuction()

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {

		if currentAuction != nil {
			auctionTxt, _ := templates.Process("auctions/auction-update", currentAuction, user.UserId)
			user.SendTextLegacy(auctionTxt)
		} else {
			user.SendTextLegacy(`No current auctions. You can auction something, though!`)
		}
		return true, nil
	}

	if args[0] == `history` {

		headers := []string{"Date", "Item", "Seller", "Buyer", "Winning Bid"}
		formatting := []string{
			`<ansi fg="magenta">%s</ansi>`,
			`<ansi fg="item">%s</ansi>`,
			`<ansi fg="username">%s</ansi>`,
			`<ansi fg="username">%s</ansi>`,
			`<ansi fg="gold">%s</ansi>`,
		}

		rows := [][]string{}

		auctionHistory := mod.auctionMgr.GetAuctionHistory(0)

		for i := len(auctionHistory) - 1; i >= 0; i-- {
			aItem := auctionHistory[i]

			buyerName := aItem.BuyerName
			sellerName := aItem.SellerName
			if aItem.Anonymous {
				buyerName = `Anonymous`
				sellerName = `Anonymous`
			}
			rows = append(rows, []string{
				aItem.EndTime.Format("2006-01-02 15:04:05"),
				aItem.ItemName,
				sellerName,
				buyerName,
				strconv.Itoa(aItem.WinningBid) + " gold",
			})
		}

		historyTableData := templates.GetTable(`Past Auctions`, headers, rows, formatting)

		tplTxt, _ := templates.Process("tables/generic", historyTableData, user.UserId)
		user.SendTextLegacy(tplTxt)

		return true, nil
	}

	if args[0] == `bid` {

		if currentAuction == nil {
			user.SendTextLegacy(`There is not an auction to bid on.`)
			return true, nil
		}

		if currentAuction.SellerUserId == user.UserId {
			user.SendTextLegacy(`You cannot bid on your own auction.`)
			return true, nil
		}

		if currentAuction.HighestBidUserId == user.UserId {
			user.SendTextLegacy(`You are already the highest bidder.`)
			return true, nil
		}

		if len(args) < 2 {
			user.SendTextLegacy(`Bid how much?`)
			return true, nil
		}

		minBid := currentAuction.HighestBid + 1
		if minBid == 0 {
			minBid = currentAuction.MinimumBid
		}

		amt, _ := strconv.Atoi(args[1])
		if amt < minBid {
			user.SendTextLegacy(fmt.Sprintf(`You must bid at least <ansi fg="gold">%d gold</ansi>.`, minBid))
			return true, nil
		}

		if amt > user.Character.Gold {
			user.SendTextLegacy(`You don't have that much gold.`)
			return true, nil
		}

		if err := mod.auctionMgr.Bid(user.UserId, amt); err != nil {
			user.SendTextLegacy(err.Error())
			return true, nil
		}

		user.Character.Gold -= amt

		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			GoldChange: -amt,
		})

		// Broadcast the bid
		auctionTxt, _ := templates.Process("auctions/auction-bid", currentAuction, user.UserId)
		for _, uid := range users.GetOnlineUserIds() {
			if u := users.GetByUserId(uid); u != nil {
				auctionOn := u.GetConfigOption(`auction`)
				if auctionOn == nil || auctionOn.(bool) {
					u.SendTextLegacy(auctionTxt)
				}
			}
		}

		sellerName := currentAuction.SellerName
		buyerName := currentAuction.HighestBidderName
		if currentAuction.Anonymous {
			sellerName = `Someone`
			buyerName = `Someone`
		}

		events.AddToQueue(AuctionUpdate{
			State:           `BID`,
			ItemName:        currentAuction.ItemData.NameComplex(),
			ItemDescription: currentAuction.ItemData.GetSpec().Description,
			SellerName:      sellerName,
			BuyerName:       buyerName,
			BidAmount:       currentAuction.HighestBid,
		})

		return true, nil
	}

	// If there is already an auction happening, abort this attempt.
	if currentAuction != nil {
		user.SendTextLegacy(`There is already an auction in progress.`)
		return true, nil
	}

	// Check whether the user has an item in their inventory that matches
	matchItem, found := user.Character.FindInBackpack(rest)

	if !found {
		user.SendTextLegacy(fmt.Sprintf("You don't have a %s to auction.", rest))
		return true, nil
	}

	cmdPrompt, _ := user.StartPrompt(`auction`, rest)
	questionConfirm := cmdPrompt.Ask(`Auction your `+matchItem.NameComplex()+`?`, []string{`Yes`, `No`})
	if !questionConfirm.Done {
		return true, nil
	}

	if questionConfirm.Response != `Yes` {
		user.SendTextLegacy(`Aborting auction`)
		user.ClearPrompt()
		return true, nil
	}

	questionAmount := cmdPrompt.Ask(`Auction for how much gold?`, []string{})
	if !questionAmount.Done {
		return true, nil
	}

	amt, _ := strconv.Atoi(questionAmount.Response)
	if amt < 1 {
		user.SendTextLegacy(`Aborting auction`)
		user.ClearPrompt()
		return true, nil
	}

	user.ClearPrompt()

	user.SendTextLegacy(fmt.Sprintf("Auctioning your <ansi fg=\"item\">%s</ansi> for <ansi fg=\"gold\">%d gold</ansi>.", matchItem.DisplayName(), amt))

	duration := 60
	if dur, ok := mod.plug.Config.Get(`DurationSeconds`).(int); ok {
		duration = dur
	}

	anonymous := false
	if anon, ok := mod.plug.Config.Get(`Anonymous`).(bool); ok {
		anonymous = anon
	}

	if mod.auctionMgr.StartAuction(matchItem, user.UserId, amt, duration, anonymous) {
		user.Character.RemoveItem(matchItem)

		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   matchItem,
			Gained: false,
		})

	}

	return true, nil
}

func (mod *AuctionsModule) newRoundHandler(e events.Event) events.ListenerReturn {

	evt := e.(events.NewRound)

	auctionNow := mod.auctionMgr.GetCurrentAuction()
	if auctionNow == nil {
		return events.Continue
	}

	if auctionNow.IsEnded() {

		mod.auctionMgr.EndAuction()

		auctionNow.LastUpdate = evt.TimeNow

		for _, uid := range users.GetOnlineUserIds() {

			auctionTxt, _ := templates.Process("auctions/auction-end", auctionNow, uid)

			if u := users.GetByUserId(uid); u != nil {
				auctionOn := u.GetConfigOption(`auction`)
				if auctionOn == nil || auctionOn.(bool) {
					u.SendTextLegacy(auctionTxt)
				}
			}
		}

		// Give the item to the winner and let them know
		if auctionNow.HighestBidUserId > 0 {

			if user := users.GetByUserId(auctionNow.HighestBidUserId); user != nil {
				if user.Character.StoreItem(auctionNow.ItemData) {

					events.AddToQueue(events.ItemOwnership{
						UserId: user.UserId,
						Item:   auctionNow.ItemData,
						Gained: true,
					})

					msg := fmt.Sprintf(`<ansi fg="yellow">You have won the auction for the <ansi fg="item">%s</ansi>! It has been added to your backpack.</ansi>`, auctionNow.ItemData.DisplayName())
					user.SendTextLegacy(msg)
				}
			} else {

				msg := fmt.Sprintf(`You won the auction for the <ansi fg="item">%s</ansi> while you were offline.`, auctionNow.ItemData.DisplayName())

				users.SearchOfflineUsers(func(u *users.UserRecord) bool {
					if u.UserId == auctionNow.HighestBidUserId {
						user = u
						return false
					}
					return true
				})

				if user != nil {
					user.Inbox.Add(
						users.Message{
							FromName: `Auction System`,
							Message:  msg,
							Item:     &auctionNow.ItemData,
						},
					)
					users.SaveUser(*user)
				}

			}

			if auctionNow.SellerUserId > 0 {

				msg := fmt.Sprintf(`Your auction of the <ansi fg="item">%s</ansi> has ended. The highest bid was made by <ansi fg="username">%s</ansi> for <ansi fg="gold">%d gold</ansi>.`, auctionNow.ItemData.DisplayName(), auctionNow.HighestBidderName, auctionNow.HighestBid)

				if sellerUser := users.GetByUserId(auctionNow.SellerUserId); sellerUser != nil {
					sellerUser.Character.Bank += auctionNow.HighestBid
					sellerUser.SendTextLegacy(`<ansi fg="yellow">` + msg + `</ansi>`)

					events.AddToQueue(events.EquipmentChange{
						UserId:     sellerUser.UserId,
						BankChange: auctionNow.HighestBid,
					})

				} else {

					msg := fmt.Sprintf(`Your auction of the <ansi fg="item">%s</ansi> has ended while you were offline. The highest bid was made by <ansi fg="username">%s</ansi> for <ansi fg="gold">%d gold</ansi>.`, auctionNow.ItemData.DisplayName(), auctionNow.HighestBidderName, auctionNow.HighestBid)

					users.SearchOfflineUsers(func(u *users.UserRecord) bool {
						if u.UserId == auctionNow.SellerUserId {
							sellerUser = u
							return false
						}
						return true
					})

					if sellerUser != nil {
						sellerUser.Inbox.Add(
							users.Message{
								FromName: `Auction System`,
								Message:  msg,
								Gold:     auctionNow.HighestBid,
								Item:     &auctionNow.ItemData,
							},
						)
						users.SaveUser(*sellerUser)
					}

				}
			}

		} else if auctionNow.SellerUserId > 0 {
			if user := users.GetByUserId(auctionNow.SellerUserId); user != nil {
				if user.Character.StoreItem(auctionNow.ItemData) {

					events.AddToQueue(events.ItemOwnership{
						UserId: user.UserId,
						Item:   auctionNow.ItemData,
						Gained: true,
					})

					msg := fmt.Sprintf(`<ansi fg="yellow">The auction for the <ansi fg="item">%s</ansi> has ended without a winner. It has been returned to you.</ansi>`, auctionNow.ItemData.DisplayName())
					user.SendTextLegacy(msg)
				}
			}

			for _, uid := range users.GetOnlineUserIds() {
				if uid == auctionNow.SellerUserId {
					continue
				}
				if u := users.GetByUserId(uid); u != nil {
					auctionOn := u.GetConfigOption(`auction`)
					if auctionOn == nil || auctionOn.(bool) {
						msg := fmt.Sprintf(`<ansi fg="yellow">The auction for the <ansi fg="item">%s</ansi> has ended without a winner. It has been returned to the seller.</ansi>`, auctionNow.ItemData.DisplayName())
						u.SendTextLegacy(msg)
					}
				}
			}

		}

		sellerName := auctionNow.SellerName
		buyerName := auctionNow.HighestBidderName
		if auctionNow.Anonymous {
			sellerName = `(Anonymous)`
			buyerName = `(Anonymous)`
		}

		events.AddToQueue(AuctionUpdate{
			State:           `END`,
			ItemName:        auctionNow.ItemData.NameComplex(),
			ItemDescription: auctionNow.ItemData.GetSpec().Description,
			SellerName:      sellerName,
			BuyerName:       buyerName,
			BidAmount:       auctionNow.HighestBid,
		})

	} else if auctionNow.LastUpdate.IsZero() {

		auctionNow.LastUpdate = evt.TimeNow

		for _, uid := range users.GetOnlineUserIds() {

			auctionTxt, _ := templates.Process("auctions/auction-start", auctionNow, uid)

			if u := users.GetByUserId(uid); u != nil {
				auctionOn := u.GetConfigOption(`auction`)
				if auctionOn == nil || auctionOn.(bool) {
					u.SendTextLegacy(auctionTxt)
				}
			}
		}

		sellerName := auctionNow.SellerName
		buyerName := auctionNow.HighestBidderName
		if auctionNow.Anonymous {
			sellerName = `(Anonymous)`
			buyerName = `(Anonymous)`
		}

		events.AddToQueue(AuctionUpdate{
			State:           `START`,
			ItemName:        auctionNow.ItemData.NameComplex(),
			ItemDescription: auctionNow.ItemData.GetSpec().Description,
			SellerName:      sellerName,
			BuyerName:       buyerName,
			BidAmount:       auctionNow.HighestBid,
		})

	} else if time.Since(auctionNow.LastUpdate) > time.Second*time.Duration(mod.plug.Config.Get(`UpdateSeconds`).(int)) {

		auctionNow.LastUpdate = evt.TimeNow

		for _, uid := range users.GetOnlineUserIds() {

			auctionTxt, _ := templates.Process("auctions/auction-update", auctionNow, uid)

			if u := users.GetByUserId(uid); u != nil {
				auctionOn := u.GetConfigOption(`auction`)
				if auctionOn == nil || auctionOn.(bool) {
					u.SendTextLegacy(auctionTxt)
				}
			}
		}

		sellerName := auctionNow.SellerName
		buyerName := auctionNow.HighestBidderName
		if auctionNow.Anonymous {
			sellerName = `(Anonymous)`
			buyerName = `(Anonymous)`
		}

		events.AddToQueue(AuctionUpdate{
			State:           `REMINDER`,
			ItemName:        auctionNow.ItemData.NameComplex(),
			ItemDescription: auctionNow.ItemData.GetSpec().Description,
			SellerName:      sellerName,
			BuyerName:       buyerName,
			BidAmount:       auctionNow.HighestBid,
		})

	}

	return events.Continue
}

type AuctionManager struct {
	ActiveAuction   *AuctionItem `yaml:"ActiveAuction,omitempty"`
	maxHistoryItems int
	PastAuctions    []PastAuctionItem `yaml:"PastAuctions,omitempty"`
}

type AuctionItem struct {
	ItemData          items.Item
	SellerUserId      int
	SellerName        string
	Anonymous         bool
	EndTime           time.Time
	MinimumBid        int
	HighestBid        int
	HighestBidUserId  int
	HighestBidderName string
	LastUpdate        time.Time
}

type PastAuctionItem struct {
	ItemName   string
	WinningBid int
	Anonymous  bool
	SellerName string
	BuyerName  string
	EndTime    time.Time
}

func (a *AuctionItem) IsEnded() bool {
	return time.Now().After(a.EndTime)
}

func (am *AuctionManager) StartAuction(item items.Item, userId int, minimumBid int, durationSeconds int, anon bool) bool {

	if am.ActiveAuction != nil {
		return false
	}

	if u := users.GetByUserId(userId); u != nil {
		am.ActiveAuction = &AuctionItem{
			ItemData:          item,
			SellerUserId:      userId,
			SellerName:        u.Character.Name,
			Anonymous:         anon,
			EndTime:           time.Now().Add(time.Second * time.Duration(durationSeconds)),
			MinimumBid:        minimumBid,
			HighestBid:        0,
			HighestBidUserId:  0,
			HighestBidderName: ``,
		}

		return true
	}

	return false
}

func (am *AuctionManager) GetCurrentAuction() *AuctionItem {
	return am.ActiveAuction
}

func (am *AuctionManager) Bid(userId int, bid int) error {

	if am.ActiveAuction == nil {
		return errors.New("There is not an auction to bid on.")
	}

	if am.ActiveAuction.HighestBidUserId == userId {
		return errors.New("You are already the highest bidder.")
	}

	if bid < am.ActiveAuction.MinimumBid || bid < am.ActiveAuction.HighestBid+1 {
		minBid := am.ActiveAuction.MinimumBid
		if am.ActiveAuction.HighestBid > 0 {
			minBid = am.ActiveAuction.HighestBid + 1
		}
		return fmt.Errorf(`The minimum bid is <ansi fg="gold">%d gold</ansi>`, minBid)
	}

	u := users.GetByUserId(userId)
	if u == nil {
		return errors.New("User not found.")
	}

	am.ActiveAuction.HighestBid = bid
	am.ActiveAuction.HighestBidUserId = userId
	am.ActiveAuction.HighestBidderName = u.Character.Name

	return nil
}

func (am *AuctionManager) EndAuction() {

	if am.ActiveAuction == nil {
		return
	}

	if am.ActiveAuction.HighestBidUserId != 0 {

		am.PastAuctions = append(am.PastAuctions, PastAuctionItem{
			ItemName:   am.ActiveAuction.ItemData.NameComplex(),
			WinningBid: am.ActiveAuction.HighestBid,
			Anonymous:  am.ActiveAuction.Anonymous,
			SellerName: am.ActiveAuction.SellerName,
			BuyerName:  am.ActiveAuction.HighestBidderName,
			EndTime:    am.ActiveAuction.EndTime,
		})

		for len(am.PastAuctions) > am.maxHistoryItems {
			am.PastAuctions = am.PastAuctions[1:]
		}

	}

	am.ActiveAuction = nil

}

func (am *AuctionManager) GetAuctionHistory(totalItems int) []PastAuctionItem {

	if totalItems < 1 {
		return am.PastAuctions
	}

	if totalItems > len(am.PastAuctions) {
		totalItems = len(am.PastAuctions)
	}

	return am.PastAuctions[len(am.PastAuctions)-totalItems : totalItems]
}

func (am *AuctionManager) GetLastAuction() PastAuctionItem {
	if len(am.PastAuctions) == 0 {
		return PastAuctionItem{}
	}

	return am.PastAuctions[len(am.PastAuctions)-1]
}
