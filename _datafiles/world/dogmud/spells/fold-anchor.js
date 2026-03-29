// Fold Anchor spell script — sets a Chrysalis anchor at current location

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(),
        'You weave a Chrysalis anchor into the fabric of this location.');
    SendRoomMessage(sourceActor.GetRoomId(),
        sourceActor.GetCharacterName(true) +
        ' traces a complex pattern in the air that briefly glows and fades.',
        sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(),
        'The anchor takes shape, binding to the Veil...');
}

function onMagic(sourceActor, targetActor) {
    var currentRoom = sourceActor.GetRoomId();
    sourceActor.SetMiscCharacterData('fold-anchor-room', currentRoom);
    SendUserMessage(sourceActor.UserId(),
        'A Chrysalis anchor locks into place here. ' +
        'Cast <ansi fg="command">fold-recall</ansi> from elsewhere to return.');
    SendRoomMessage(currentRoom,
        'A faint shimmer marks where ' +
        sourceActor.GetCharacterName(true) +
        ' has set an anchor.',
        sourceActor.UserId());
}
