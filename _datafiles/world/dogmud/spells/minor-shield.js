// Minor Shield spell script — flavor only; effects resolved in Go (Stage 11.4)

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You weave arcane energy into a protective barrier.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' weaves a shimmer of energy around them.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
