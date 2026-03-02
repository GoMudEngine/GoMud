// Abandoned Campsite - Room 407
// Grants quest step 4-investigate when a player on quest 4 enters.

function onEnter(user, room) {

    if ( user.HasQuest("4-start") && !user.HasQuest("4-investigate") ) {
        user.GiveQuest("4-investigate");
        user.SendText('The scattered debris and fresh boot prints confirm it -- this is where the bandits have been operating. There may be more evidence deeper in the gully.');
    }

    return true;
}
