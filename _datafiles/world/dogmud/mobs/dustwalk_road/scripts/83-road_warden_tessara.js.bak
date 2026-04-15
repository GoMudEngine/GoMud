// Road Warden Tessara - Mob 83
// Handles quest item delivery for Quest 4 (The Warden's Report).
// When a player gives the bandit's purse (item 16), auto-advance the quest.

const PURSE_ITEM_ID = 16;

function onGive(mob, room, eventDetails) {

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) {
        return false;
    }

    // Only care about item gives, not gold
    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    // If they gave the bandit's purse and are on quest step 4-start
    if ( eventDetails.item.ItemId == PURSE_ITEM_ID
        && user.HasQuest("4-start")
        && !user.HasQuest("4-end") ) {

        mob.Command('say Caravan schedules. Supply routes.', 1.0);
        mob.Command('say This is not petty theft -- someone is feeding them information from inside the trading houses.', 2.5);
        mob.Command('say This changes things. You have done good work. The Watchers will act on this.', 4.0);
        user.GiveQuest("4-end");
        return true;
    }

    // Not the right item — give it back
    mob.Command('say I appreciate the thought, but I do not need that.');
    mob.Command('give !' + String(eventDetails.item.ItemId) + ' ' + user.ShorthandId());
    return true;
}
