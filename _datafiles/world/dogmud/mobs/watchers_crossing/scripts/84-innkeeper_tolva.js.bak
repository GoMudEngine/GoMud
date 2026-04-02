// Innkeeper Tolva - Mob 84
// Handles give-based quest item delivery for Quest 5 (The Innkeeper's Complaint).
// Player may give the crossing ledger instead of asking about it.

const LEDGER_ITEM_ID = 21;

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    var itemId = eventDetails.item.ItemId;

    if ( itemId == LEDGER_ITEM_ID ) {
        if ( !user.HasQuest("5-start") ) {
            mob.Command('say A ledger? I do not need a ledger. But if you want to help, ask me about it.');
            return true;
        }

        if ( user.HasQuest("5-end") ) {
            mob.Command('say I already have what I need. Thank you.');
            return true;
        }

        user.GiveQuest("5-end");
        mob.Command('say The crossing ledger! Let me see...', 1.0);
        mob.Command('say No commission. No charter number. No authorizing signature. Just Harn collecting and spending as he sees fit. This is exactly what I needed. The magistrate in Thornwall will see this before the month is out.', 3.0);
        return true;
    }

    mob.Command('say Thank you, but I do not need that. Can I get you something from the kitchen?');
    return true;
}
