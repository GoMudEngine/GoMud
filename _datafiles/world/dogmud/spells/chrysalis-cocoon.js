// Chrysalis Cocoon spell script — flavor only; effects resolved in Go

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You draw living Chrysalis around the target, weaving a protective cocoon.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' gestures as tendrils of living Chrysalis weave into a shell.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
