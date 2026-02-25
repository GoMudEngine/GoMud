// Hemorrhagic Burst spell script — flavor only; effects resolved in Go (Stage 11.4)

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You focus on the blood of your enemies, willing it to boil and rupture outward!');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' clenches their fists, veins pulsing with terrible intent.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
