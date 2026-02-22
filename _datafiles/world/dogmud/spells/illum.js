
// Called when the casting is initialized (cast command) — only used for mob casts.
// Player cast initiation messages are handled by skill.cast.go.
function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You begin focusing a point of light into existence.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' begins to concentrate, eyes half-closed.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You hold the image steady, splitting it again...');
}

// onMagic: not called by the player-cast resolution system.
// Buff application is handled by effect_type: buff / buff_ids in illum.yaml.
function onMagic(sourceActor, targetActor) {
    return true;
}
