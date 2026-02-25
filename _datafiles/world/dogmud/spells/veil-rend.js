// Veil Rend spell script — flavor only; effects resolved in Go

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You reach deep into the Veil and begin tearing it apart!');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' raises both arms as reality itself begins to crack and splinter!', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
