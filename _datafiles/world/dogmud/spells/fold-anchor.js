// Fold Anchor spell script — sets a recall point via MiscCharacterData

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You weave a Chrysalis anchor into the fabric of this location.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' traces a complex pattern in the air that briefly glows and fades.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {
    var roomId = sourceActor.GetRoomId();
    sourceActor.SetMiscCharacterData('fold-anchor-room', roomId.toString());
    SendUserMessage(sourceActor.UserId(), 'A Chrysalis anchor locks into place here. You can use Fold Recall to return.');
    SendRoomMessage(sourceActor.GetRoomId(), 'A faint shimmer marks where '+sourceActor.GetCharacterName(true)+' has set an anchor.', sourceActor.UserId());
}
