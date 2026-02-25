// Neural Toxin spell script — flavor only; effects resolved in Go

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You channel psionic venom toward your target\'s mind.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' focuses with toxic intent, a sickly glow around their hands.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
