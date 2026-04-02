// The Tollhouse - Room 421
// Quest item pickup: crossing ledger (item 21) for Q5
// Player can take/get the ledger if on quest step 5-start

var LEDGER_ITEM_ID = 21;

function onCommand(cmd, rest, user, room) {

    if ( (cmd == "get" || cmd == "take" || cmd == "grab") && rest != null && rest != "" ) {
        var target = rest.toLowerCase();
        if ( target == "ledger" || target == "heavy ledger" || target == "crossing ledger" ) {

            // Must be on Q5
            if ( !user.HasQuest("5-start") ) {
                user.SendText('The heavy ledger sits on the counter behind the booth. It is not yours to take.');
                return true;
            }

            // Already picked it up
            if ( user.HasItemId(LEDGER_ITEM_ID) ) {
                user.SendText('You already have the crossing ledger.');
                return true;
            }

            // Already completed the evidence step
            if ( user.HasQuest("5-evidence") ) {
                user.SendText('You have already taken what you needed from the ledger.');
                return true;
            }

            user.GiveItem(LEDGER_ITEM_ID);
            user.GiveQuest("5-ledger");
            user.GiveQuest("5-evidence");
            user.SendText('You slip the heavy ledger off the counter while the toll collector is distracted. The pages are filled with dates, amounts, and names -- but nowhere do you see an official charter number or authorizing signature. This is all the proof Tolva needs.');
            room.SendText(user.GetCharacterName(true)+' slips a heavy ledger off the counter.', user.UserId());
            return true;
        }
    }

    return false;
}
