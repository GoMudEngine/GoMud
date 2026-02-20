// Fireball spell script — flavor only; effects resolved in Go (Stage 11.4)

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You pull heat from the air and compress it into a crackling sphere!');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' draws the heat from the air, forming a searing orb.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
