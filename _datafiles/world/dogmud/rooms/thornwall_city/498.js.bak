// Operations Room - Room 498
// 1. Grants 14-confront when player enters with 14-explore
// 2. Intercepts strongbox interaction -- requires key (40034) to get ledger

var STRONGBOX_KEY_ID = 40034;
var BRIBE_LEDGER_ID = 40036;

function onCommand(cmd, rest, user, room) {

    // Strongbox interaction
    if ( (cmd == "open" || cmd == "unlock" || cmd == "get" || cmd == "search" || cmd == "look" || cmd == "use") && rest != null && rest != "" ) {
        var target = rest.toLowerCase();
        if ( target == "strongbox" || target == "box" || target == "lockbox" ) {

            // Already got the ledger
            if ( user.HasQuest("14-evidence") ) {
                user.SendText('The strongbox is open and empty. You have already taken the ledger.');
                return true;
            }

            // Need the quest active
            if ( !user.HasQuest("14-confront") ) {
                user.SendText('A heavy iron strongbox beneath the table, secured with a brass padlock. You have no reason to open it.');
                return true;
            }

            // Check for key
            if ( !user.HasItemId(STRONGBOX_KEY_ID) ) {
                user.SendText('The strongbox is locked with a heavy brass padlock. You need a key.');
                return true;
            }

            // Has key -- open strongbox, consume key, give ledger, grant quest
            var items = user.GetBackpackItems();
            for (var i = 0; i < items.length; i++) {
                if (items[i].ItemId() === STRONGBOX_KEY_ID) {
                    user.TakeItem(items[i]);
                    break;
                }
            }
            user.GiveItem(BRIBE_LEDGER_ID);
            user.GiveQuest("14-evidence");
            user.SendText('You fit the brass key into the padlock and turn. The lock clicks open. Inside the strongbox you find a slim leather journal -- a bribe ledger, filled with names, dates, and amounts paid to corrupt city officials. This is the evidence you need.');
            room.SendText(user.GetCharacterName(true)+' unlocks the strongbox and pulls out a leather journal, eyes widening at its contents.', user.UserId());
            return true;
        }
    }

    return false;
}

function onEnter(user, room) {

    // Grant 14-confront when entering with 14-explore but not 14-confront
    if ( user.HasQuest("14-explore") && !user.HasQuest("14-confront") ) {
        user.GiveQuest("14-confront");
        user.SendText('You step into a well-lit chamber that serves as the nerve center of the smuggling operation. A man at the table looks up sharply -- this must be the one in charge. A heavy iron strongbox sits beneath his table, locked tight. Whatever proof you need is probably in there.');
    }

    // Must return true so the engine fires the auto-look
    return true;
}
