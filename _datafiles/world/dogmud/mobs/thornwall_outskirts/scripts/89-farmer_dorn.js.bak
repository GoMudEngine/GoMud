// Farmer Dorn - Mob 89
// Handles quest item delivery for Quest 7 (The Fallow Field).
// When a player gives the pest sample (item 24), auto-advance the quest.

const PEST_SAMPLE_ID = 24;

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    // Only care about item gives, not gold
    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    // Pest sample delivery for Q7
    if ( eventDetails.item.ItemId == PEST_SAMPLE_ID ) {
        if ( user.HasQuest("7-start") && !user.HasQuest("7-end") ) {
            mob.Command('say Too big. This is not natural size.', 1.0);
            mob.Command('say Something in the fallow soil is feeding them, making them grow. As long as that field sits empty, this will keep happening.', 2.5);
            mob.Command('say You have done what you can. The rest is politics, and politics requires getting inside those walls.', 4.5);
            user.GiveQuest("7-end");
            return true;
        }
        // Has the sample but wrong quest state
        mob.Command('say I do not need that. Not now.');
        return true;
    }

    // Not the right item — give it back
    mob.Command('say I appreciate the thought, but what would I do with that?');
    mob.Command('give !' + String(eventDetails.item.ItemId) + ' ' + user.ShorthandId());
    return true;
}
