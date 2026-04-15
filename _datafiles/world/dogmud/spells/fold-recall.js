// Fold Recall spell script — teleport to your Chrysalis anchor

function onCast(sourceActor, targetActor) {
    var currentRoomId = sourceActor.GetRoomId();
    var currentRoom = GetRoom(currentRoomId);

    // Check if recall is blocked in this room (instanced zones with allow_recall: false)
    if (currentRoom && currentRoom.GetTempData('allow_recall') === false) {
        SendUserMessage(sourceActor.UserId(),
            'Something about this place prevents you from recalling.');
        return false;
    }

    var anchorRoom = Number(
        sourceActor.GetMiscCharacterData('fold-anchor-room')) || 0;

    if (anchorRoom <= 0) {
        SendUserMessage(sourceActor.UserId(),
            'You reach for the Veil, but there is no anchor to ' +
            'pull you. Set one first with ' +
            '<ansi fg="command">cast fold-anchor</ansi>.');
        return false;
    }

    if (anchorRoom == currentRoomId) {
        SendUserMessage(sourceActor.UserId(),
            'You are already standing on your anchor.');
        return false;
    }

    return true;
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
