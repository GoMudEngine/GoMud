

function onIdle(room) {
    
    if ( room.AddTemporaryExit('shimmering portal', ':cyan', 0, '15 minutes') ) {
        room.SendText('A portal to the world of the living appears!');
    }
    return false;
}

function onExit(user , room) {
    // Remove the healing buff if they are leaving
    user.RemoveBuff(24);
}
