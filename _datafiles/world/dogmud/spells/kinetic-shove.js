// Kinetic Shove spell script — flavor only; effects resolved in Go

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You gather telekinetic force and thrust it forward.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' thrusts their palm forward, the air distorting.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
