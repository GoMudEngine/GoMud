// Fold Recall spell script — teleport to your Chrysalis anchor

function onCast(sourceActor, targetActor) {
    var anchorRoom = Number(
        sourceActor.GetMiscCharacterData('fold-anchor-room')) || 0;

    if (anchorRoom <= 0) {
        SendUserMessage(sourceActor.UserId(),
            'You reach for the Veil, but there is no anchor to ' +
            'pull you. Set one first with ' +
            '<ansi fg="command">cast fold-anchor</ansi>.');
        return false;
    }

    var currentRoom = sourceActor.GetRoomId();
    if (anchorRoom == currentRoom) {
        SendUserMessage(sourceActor.UserId(),
            'You are already standing on your anchor.');
        return false;
    }

    SendUserMessage(sourceActor.UserId(),
        'You reach through the Veil toward your anchor point...');
    SendRoomMessage(currentRoom,
        sourceActor.GetCharacterName(true) +
        ' reaches into the Veil, reality blurring around them.',
        sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(),
        'The Veil thins as you pull yourself toward the anchor...');
}

function onMagic(sourceActor, targetActor) {
    var anchorRoom = Number(
        sourceActor.GetMiscCharacterData('fold-anchor-room')) || 0;
    var currentRoom = sourceActor.GetRoomId();

    if (anchorRoom <= 0 || anchorRoom == currentRoom) {
        SendUserMessage(sourceActor.UserId(),
            'The fold collapses — no valid anchor found.');
        return;
    }

    // Clear combat state before teleporting
    sourceActor.EndCombat();

    SendRoomMessage(currentRoom,
        sourceActor.GetCharacterName(true) +
        ' folds through the Veil and vanishes!',
        sourceActor.UserId());
    sourceActor.MoveRoom(anchorRoom);
    SendUserMessage(sourceActor.UserId(),
        'You fold through the Veil and arrive at your anchor point!');
    SendRoomMessage(anchorRoom,
        sourceActor.GetCharacterName(true) +
        ' folds through the Veil and appears!',
        sourceActor.UserId());
}
