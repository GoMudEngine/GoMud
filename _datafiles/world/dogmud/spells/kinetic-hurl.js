// Kinetic Hurl spell script — flavor only; effects resolved in Go (Stage 11.4)

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You gather telekinetic force, the air between your hands distorting.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' gathers telekinetic force, the air distorting around their hands.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
