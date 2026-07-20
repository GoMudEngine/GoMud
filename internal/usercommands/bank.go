package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Bank(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	user.SendText(messaging.CategorySystem, ``)

	if !room.IsBank {
		user.SendText(messaging.CategorySystem, `You are not at a bank.`+term.CRLFStr)
		return true, nil
	}

	if rest == `` {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You have <ansi fg="gold">%d gold</ansi> on hand and <ansi fg="gold">%d gold</ansi> in the bank.`, user.Character.Gold, user.Character.Bank))
		user.SendText(messaging.CategorySystem, `You can <ansi fg="command">deposit</ansi> to or <ansi fg="command">withdraw</ansi> from the bank.`+term.CRLFStr)
		return true, nil
	}

	if rest == `deposit` || rest == `withdraw` {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`%s how much? Make sure to include the amount of gold or "all".%s`, rest, term.CRLFStr))
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 2 || (args[0] != `deposit` && args[0] != `withdraw`) {
		user.SendText(messaging.CategorySystem, `Try <ansi fg="command">help bank</ansi> for more information about banking.`+term.CRLFStr)
		return true, nil
	}

	action := args[0]
	amountStr := args[1]
	amount, _ := strconv.Atoi(amountStr)

	if amount < 1 && amountStr != `all` {

		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You must specify an amount greater than zero to %s.%s`, action, term.CRLFStr))
		return true, nil

	} else if action == `deposit` {
		if amountStr == `all` {
			amount = user.Character.Gold
		}

		if amount > user.Character.Gold {
			amount = user.Character.Gold
			user.SendText(messaging.CategorySystem, `You don't have that much gold on hand, but what you do have you deposit.`)
		}

		user.Character.Gold -= amount
		user.Character.Bank += amount

		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			GoldChange: -amount,
			BankChange: amount,
		})

		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You deposit <ansi fg="gold">%d gold</ansi>.`, amount))
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You now have <ansi fg="gold">%d gold</ansi> on hand and <ansi fg="gold">%d gold</ansi> in the bank.`, user.Character.Gold, user.Character.Bank))

	} else if action == `withdraw` {
		if amountStr == `all` {
			amount = user.Character.Bank
		}

		if amount > user.Character.Bank {
			amount = user.Character.Bank
			user.SendText(messaging.CategorySystem, `You don't have that much gold in the bank, but you withdraw what is there.`)
		}

		user.Character.Bank -= amount
		user.Character.Gold += amount

		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			GoldChange: -amount,
			BankChange: amount,
		})

		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You withdraw <ansi fg="gold">%d gold</ansi>.`, amount))
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You now have <ansi fg="gold">%d gold</ansi> on hand and <ansi fg="gold">%d gold</ansi> in the bank.`, user.Character.Gold, user.Character.Bank))
	}

	user.SendText(messaging.CategorySystem, ``)
	return true, nil
}
