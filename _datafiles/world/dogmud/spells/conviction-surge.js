// Conviction Surge spell script — flavor only; effects resolved in Go

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You channel conviction into empowering energy.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' gathers conviction, a fierce glow building.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
