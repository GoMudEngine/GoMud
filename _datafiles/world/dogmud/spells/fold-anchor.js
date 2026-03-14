// Fold Anchor spell script — toggle: sets anchor or recalls to it

function onCast(sourceActor, targetActor) {
    var anchorRoom = sourceActor.GetMiscCharacterData('fold-anchor-room');
    var currentRoom = sourceActor.GetRoomId();

    if (anchorRoom && anchorRoom !== '' && anchorRoom !== '0' && parseInt(anchorRoom) !== currentRoom) {
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
    var anchorRoom = sourceActor.GetMiscCharacterData('fold-anchor-room');
    var currentRoom = sourceActor.GetRoomId();

    if (anchorRoom && anchorRoom !== '' && anchorRoom !== '0' && parseInt(anchorRoom) !== currentRoom) {
        SendUserMessage(sourceActor.UserId(), 'The Veil thins as you pull yourself toward the anchor...');
    }
}

function onMagic(sourceActor, targetActor) {
    var anchorRoom = sourceActor.GetMiscCharacterData('fold-anchor-room');
    var currentRoom = sourceActor.GetRoomId();

    if (anchorRoom && anchorRoom !== '' && anchorRoom !== '0' && parseInt(anchorRoom) !== currentRoom) {
        // Recall to anchor
        var roomId = parseInt(anchorRoom);
        if (roomId < 1) {
            SendUserMessage(sourceActor.UserId(), 'Your anchor has dissipated.');
            return;
        }
        SendRoomMessage(currentRoom, sourceActor.GetCharacterName(true)+' folds through the Veil and vanishes!', sourceActor.UserId());
        sourceActor.MoveRoom(roomId);
        SendUserMessage(sourceActor.UserId(), 'You fold through the Veil and arrive at your anchor point!');
        SendRoomMessage(roomId, sourceActor.GetCharacterName(true)+' folds through the Veil and appears!', sourceActor.UserId());
    } else {
        // Set anchor
        sourceActor.SetMiscCharacterData('fold-anchor-room', currentRoom.toString());
        SendUserMessage(sourceActor.UserId(), 'A Chrysalis anchor locks into place here. Cast again from elsewhere to return.');
        SendRoomMessage(currentRoom, 'A faint shimmer marks where '+sourceActor.GetCharacterName(true)+' has set an anchor.', sourceActor.UserId());
    }
}
