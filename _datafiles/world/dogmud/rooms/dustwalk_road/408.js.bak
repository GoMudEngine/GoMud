// Bandit Hollow - Room 408
// Grants quest step 4-evidence when a player picks up the bandit's
// purse (item 16) while on quest step 4-investigate.

var PURSE_ITEM_ID = 16;

function onCommand(cmd, rest, user, room) {

    if ( cmd == "get" || cmd == "take" || cmd == "grab" || cmd == "pick" ) {
        if ( rest == null || rest == "" ) {
            return false;
        }

        var target = rest.toLowerCase();
        if ( target == "purse" || target == "bandits purse" || target == "bandit's purse" ) {

            // Only advance quest if on the right step
            if ( user.HasQuest("4-investigate") && !user.HasQuest("4-evidence") ) {
                user.GiveQuest("4-evidence");
            }
        }
    }

    return false;
}
