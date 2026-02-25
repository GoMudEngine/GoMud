// Chrysalis Regeneration spell script — flavor only; effects resolved in Go

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You call upon deep Chrysalis energy to wrap the target in regenerative power.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' draws deep Chrysalis energy, a vivid green glow pulsing.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
