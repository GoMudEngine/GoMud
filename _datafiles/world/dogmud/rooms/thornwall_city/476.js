// Records Office - Room 476
// Quest item pickup: tithe ledger (item 29) for Q9
// Player can take/get the ledger after Pell points them to the shelf

var LEDGER_ITEM_ID = 29;

function onCommand(cmd, rest, user, room) {

    if ( (cmd == "get" || cmd == "take" || cmd == "grab" || cmd == "search") && rest != null && rest != "" ) {
        var target = rest.toLowerCase();
        if ( target == "ledger" || target == "tithe ledger" || target == "shelves" || target == "shelf" ) {

            // Must be on Q9
            if ( !user.HasQuest("9-start") ) {
                user.SendText('Rows of ledgers line the shelves, each labelled by year and category. None of them are your business.');
                return true;
            }

            // Already picked it up
            if ( user.HasItemId(LEDGER_ITEM_ID) ) {
                user.SendText('You already have the tithe ledger.');
                return true;
            }

            // Already completed
            if ( user.HasQuest("9-end") ) {
                user.SendText('You have already dealt with the tithe ledger.');
                return true;
            }

            user.GiveItem(LEDGER_ITEM_ID);
            user.SendText('You find the slim leather-bound book on the shelf Pell indicated. The tithe records inside are meticulous -- and damning in both directions. Several merchants have clearly been under-reporting, and the temple\'s own expense entries show purchases that do not match inventory. Both sides are cheating.');
            room.SendText(user.GetCharacterName(true)+' pulls a ledger from the records shelf and examines it closely.', user.UserId());
            return true;
        }
    }

    return false;
}
