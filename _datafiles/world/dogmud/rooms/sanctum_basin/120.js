// Boss Cave - Room 120
// Aberrant Chrysalis -- environmental lore on entry; gives 1-cave quest on boss defeat.

const aberrantMobId = 69;

var bossEngaged = false;

function onEnter(user, room) {
    user.SendText('');
    user.SendText('The air in this chamber is wrong. The moisture on the walls has no source you can identify, and there is a pressure here that is not quite sound.');
    user.SendText('');
    user.SendText('At the far end of the cave, a shape moves. It is roughly person-shaped, but the proportions have drifted. The Chrysalis process does this sometimes -- a Becoming that lost direction. The Academy calls them Aberrants. This one does not speak. It does not need to.');
    user.SendText('');

    // Track when an eligible player enters so onIdle knows to watch for the kill
    if ( user.HasQuest("1-magic_cast") && !user.HasQuest("1-cave") ) {
        bossEngaged = true;
    }
}

function onIdle(room) {

    if ( !bossEngaged ) {
        return;
    }

    // If boss is still alive, nothing to do yet
    var boss = room.GetMob(aberrantMobId, false);
    if ( boss != null ) {
        return;
    }

    // Boss is gone -- give 1-cave to any eligible player still in the room
    var players = room.GetPlayers();
    if ( players == null ) {
        bossEngaged = false;
        return;
    }

    var anyAdvanced = false;
    for ( var i = 0; i < players.length; i++ ) {
        var player = players[i];
        if ( player.HasQuest("1-magic_cast") && !player.HasQuest("1-cave") ) {
            player.GiveQuest("1-cave");
            player.SendText('');
            player.SendText('The Aberrant collapses. The pressure in the air releases. Somewhere above, Elder Saris is probably not surprised.');
            player.SendText('The south gate is open to you now. Return to the Basin Warden.');
            player.SendText('');
            anyAdvanced = true;
        }
    }

    if ( anyAdvanced ) {
        bossEngaged = false;
    }
}

function onExit(user, room) {
}

function onLoad(room) {
    bossEngaged = false;
}
