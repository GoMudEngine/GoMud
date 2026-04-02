// Temple Priest Olen - Mob 95
// Handles quest item delivery for Quest 9 (The Temple's Tithe Audit).
// When a player gives the tithe ledger (item 29), auto-advance the quest.

const LEDGER_ITEM_ID = 29;

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    // Tithe ledger delivery for Q9
    if ( eventDetails.item.ItemId == LEDGER_ITEM_ID ) {
        if ( user.HasQuest("9-start") && !user.HasQuest("9-end") ) {
            mob.Command('say Let me see that.', 1.0);
            mob.Command('say Both sides. The merchants are under-paying, and our own expense records are... generous.', 2.5);
            mob.Command('say Thank you for your honesty. This will not be pleasant, but it needed to come to light.', 4.5);
            user.GiveQuest("9-end");
            return true;
        }
        mob.Command('say I have no need of that ledger now.');
        return true;
    }

    // Not the right item — give it back
    mob.Command('say The temple accepts tithes in coin, not goods.');
    mob.Command('give !' + String(eventDetails.item.ItemId) + ' ' + user.ShorthandId());
    return true;
}
