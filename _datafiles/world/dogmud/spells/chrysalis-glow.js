// Chrysalis Glow spell script — flavor only; buff applied by Go (Stage 11.4)

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You coax the Chrysalis spores around you to luminesce.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' concentrates as faint motes of light gather.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You hold the image steady, coaxing more spores to glow...');
}

function onMagic(sourceActor, targetActor) {
    return true;
}
