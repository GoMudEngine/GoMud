// Mend All spell script — flavor only; effects resolved in Go (Stage 11.4)

function onCast(sourceActor, targetActors) {
    SendUserMessage(sourceActor.UserId(), 'You radiate Chrysalis healing energy outward to all nearby allies.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' spreads their arms as warm energy radiates outward.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {
    SendUserMessage(sourceActor.UserId(), 'You continue channeling healing energy...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' continues radiating warm energy...', sourceActor.UserId());
}

function onMagic(sourceActor, targetActors) {}
