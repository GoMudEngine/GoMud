// Mend Flesh spell script — flavor only; effects resolved in Go (Stage 11.4)

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You reach out with Chrysalis energy, willing flesh to knit.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' extends their hands, a warm glow gathering.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You continue channeling restorative energy...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' continues channeling a warm glow...', sourceActor.UserId());
}

function onMagic(sourceActor, targetActor) {}
