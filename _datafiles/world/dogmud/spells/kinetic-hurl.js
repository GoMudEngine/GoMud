// Kinetic Hurl spell script — flavor only; effects resolved in Go (Stage 11.4)

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You grasp a stone and channel your willpower into it.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' focuses their will on a stone, which begins to vibrate.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
