// Delia - Mob 259
// Handles give-based quest completion for Quest 16.
// Player can give forest herbs (item 40040) instead of dialogue.
// CRITICAL: give.go transfers item BEFORE this script fires.

var HERBS_ITEM_ID = 40040;

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    var itemId = eventDetails.item.ItemId;

    if ( itemId == HERBS_ITEM_ID ) {

        // Player found herbs before talking to Delia
        if ( !user.HasQuest("16-start") ) {
            user.GiveQuest("16-start");
            user.GiveQuest("16-end");
            mob.Command('say Where did you find these? This is feverfew and bitter-thistle of a quality I have not seen in months.', 1.0);
            mob.Command('say You have solved a problem I have been struggling with for weeks. Thank you.', 3.0);
            return true;
        }

        // Already completed
        if ( user.HasQuest("16-end") ) {
            mob.Command('say I have what I need for now. But thank you.');
            return true;
        }

        // Normal completion
        user.GiveQuest("16-end");
        mob.Command('say These are remarkable. The color, the potency -- this is better than anything I have been able to cultivate.', 1.0);
        mob.Command('say My stock will be full again within the week. You have my gratitude.', 3.0);
        return true;
    }

    // Not the quest item -- return it
    mob.Command('say That is kind of you, but I have no use for it.');
    mob.Command('give !' + String(eventDetails.item.ItemId) + ' ' + user.ShorthandId());
    return true;
}
