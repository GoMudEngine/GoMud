// Fold Anchor spell script — toggle: sets anchor or recalls to it

function onCast(sourceActor, targetActor) {
    var anchorRoom = Number(sourceActor.GetMiscCharacterData('fold-anchor-room')) || 0;
    var currentRoom = sourceActor.GetRoomId();

    if (anchorRoom > 0 && anchorRoom != currentRoom) {
        // Anchor exists and we're somewhere else — recall mode
        SendUserMessage(sourceActor.UserId(), 'You reach through the Veil toward your anchor point...');
        SendRoomMessage(currentRoom, sourceActor.GetCharacterName(true)+' reaches into the Veil, reality blurring around them.', sourceActor.UserId());
    } else {
        // No anchor or standing on it — set/refresh mode
        SendUserMessage(sourceActor.UserId(), 'You weave a Chrysalis anchor into the fabric of this location.');
        SendRoomMessage(currentRoom, sourceActor.GetCharacterName(true)+' traces a complex pattern in the air that briefly glows and fades.', sourceActor.UserId());
    }
    return true;
}

function onWait(sourceActor, targetActor) {
    var anchorRoom = Number(sourceActor.GetMiscCharacterData('fold-anchor-room')) || 0;
    var currentRoom = sourceActor.GetRoomId();

    if (anchorRoom > 0 && anchorRoom != currentRoom) {
        SendUserMessage(sourceActor.UserId(), 'The Veil thins as you pull yourself toward the anchor...');
    }
}

function onMagic(sourceActor, targetActor) {
    var anchorRoom = Number(sourceActor.GetMiscCharacterData('fold-anchor-room')) || 0;
    var currentRoom = sourceActor.GetRoomId();

    if (anchorRoom > 0 && anchorRoom != currentRoom) {
        // Recall to anchor
        SendRoomMessage(currentRoom, sourceActor.GetCharacterName(true)+' folds through the Veil and vanishes!', sourceActor.UserId());
        sourceActor.MoveRoom(anchorRoom);
        SendUserMessage(sourceActor.UserId(), 'You fold through the Veil and arrive at your anchor point!');
        SendRoomMessage(anchorRoom, sourceActor.GetCharacterName(true)+' folds through the Veil and appears!', sourceActor.UserId());
    } else {
        // Set anchor — store as number, not string
        sourceActor.SetMiscCharacterData('fold-anchor-room', currentRoom);
        SendUserMessage(sourceActor.UserId(), 'A Chrysalis anchor locks into place here. Cast again from elsewhere to return.');
        SendRoomMessage(currentRoom, 'A faint shimmer marks where '+sourceActor.GetCharacterName(true)+' has set an anchor.', sourceActor.UserId());
    }
}
