// Peddler Malk - Mob 250
// Handles give-based quest completion for Quest 15.
// Player can give freight crate (item 40039) instead of using dialogue.
// CRITICAL: give.go transfers the item BEFORE this script fires.

var FREIGHT_CRATE_ID = 40039;

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    var itemId = eventDetails.item.ItemId;

    if ( itemId == FREIGHT_CRATE_ID ) {

        // Player found the crate before talking to Malk — grant both steps
        if ( !user.HasQuest("15-start") ) {
            user.GiveQuest("15-start");
            user.GiveQuest("15-end");
            mob.Command('say My freight! Where did you -- never mind, I can see the lock has been forced. Bandits.', 1.0);
            mob.Command('say You have done me a considerable favor. Road prices from me, any time. Better than road prices.', 3.0);
            return true;
        }

        // Already completed
        if ( user.HasQuest("15-end") ) {
            mob.Command('say You already brought my shipment back. I am in your debt.');
            return true;
        }

        // Normal completion — player had quest, delivers crate
        user.GiveQuest("15-end");
        mob.Command('say That is the one! The iron straps, the stencil marks -- that is my consignment.', 1.0);
        mob.Command('say I owe you for this. Road prices from me, any time. Better than road prices.', 3.0);
        return true;
    }

    // Not the quest item — return it
    mob.Command('say I appreciate the thought, but that is yours.');
    mob.Command('give !' + String(eventDetails.item.ItemId) + ' ' + user.ShorthandId());
    return true;
}
