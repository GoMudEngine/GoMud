// Pyretic Surge spell script — flavor only; effects resolved in Go (Stage 11.4)

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You channel Chrysalis energy, feeling the air shimmer with heat.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' holds out their hands as the air begins to ripple with heat.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
