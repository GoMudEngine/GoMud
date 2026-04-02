// Toll Collector Harn - Mob 86
// If player gives the report back to Harn, he hands it back.

const REPORT_ITEM_ID = 31;

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    var itemId = eventDetails.item.ItemId;

    if ( itemId == REPORT_ITEM_ID ) {
        // Give it back to the player
        user.GiveItem(REPORT_ITEM_ID);
        mob.Command('say No, no -- I need you to take that to Thornwall! Clerk Pell in the Records Office. He will file it with the magistrate. Here, take it back.');
        return true;
    }

    return false;
}
