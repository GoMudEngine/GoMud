// Fold Anchor spell script — sets a Chrysalis anchor at current location

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
