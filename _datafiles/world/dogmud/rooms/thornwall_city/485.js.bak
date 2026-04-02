// Tavern Cellar - Room 485
// Gates descent into the tunnels. Player must have 14-start to go down.

function onCommand(cmd, rest, user, room) {

    // Only intercept movement commands heading down
    if ( cmd != "go" && cmd != "down" ) { return false; }

    var target = "";
    if ( cmd == "go" && rest != null ) {
        target = rest.toLowerCase();
    } else if ( cmd == "down" ) {
        target = "down";
    }

    if ( target != "down" ) { return false; }

    // If player has the quest, let them pass
    if ( user.HasQuest("14-start") ) {
        return false;
    }

    // Block descent without the quest
    user.SendText('The iron grate covers a dark opening that descends into the old drainage tunnels. You have no reason to go down there.');
    return true;
}
