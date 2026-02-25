// Fold Recall spell script — teleports to fold-anchor location

function onCast(sourceActor, targetActor) {
    var anchorRoom = sourceActor.GetMiscCharacterData('fold-anchor-room');
    if (!anchorRoom || anchorRoom === '' || anchorRoom === '0') {
        SendUserMessage(sourceActor.UserId(), 'You have no Fold Anchor set. Cast Fold Anchor first.');
        return false;
    }
    SendUserMessage(sourceActor.UserId(), 'You reach through the Veil toward your anchor point...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' reaches into the Veil, reality blurring around them.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'The Veil thins as you pull yourself toward the anchor...');
}

function onMagic(sourceActor, targetActor) {
    var anchorRoom = sourceActor.GetMiscCharacterData('fold-anchor-room');
    if (!anchorRoom || anchorRoom === '' || anchorRoom === '0') {
        SendUserMessage(sourceActor.UserId(), 'Your Fold Anchor has dissipated.');
        return;
    }
    var roomId = parseInt(anchorRoom);
    if (roomId < 1) {
        SendUserMessage(sourceActor.UserId(), 'Your Fold Anchor has dissipated.');
        return;
    }

    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' folds through the Veil and vanishes!', sourceActor.UserId());
    sourceActor.MoveRoom(roomId);
    SendUserMessage(sourceActor.UserId(), 'You fold through the Veil and arrive at your anchor point!');
    SendRoomMessage(roomId, sourceActor.GetCharacterName(true)+' folds through the Veil and appears!', sourceActor.UserId());
}
