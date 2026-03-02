function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    // Only care about item gives, not gold
    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    // Healing poultice delivery for Q2
    if ( eventDetails.item.ItemId == 30010 ) {
        if ( user.HasQuest("2-start") && !user.HasQuest("2-poultices") ) {
            user.GiveQuest("2-poultices");
            mob.Command('say These are good. Strong. Our sick will benefit...', 1.0);
            mob.Command('say Go deeper into the tunnels -- the warriors will not stop you now.', 2.5);
            return true;
        }
        // Has the poultice but wrong quest state
        mob.Command('say I have no need of this.');
        return true;
    }

    // Not the right item
    mob.Command('say I have no need of this.');
    return true;
}
