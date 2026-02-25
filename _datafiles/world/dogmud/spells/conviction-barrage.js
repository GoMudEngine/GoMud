// Conviction Barrage spell script — flavor only; effects resolved in Go

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You compress your conviction into hundreds of razor-sharp fragments!');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' raises their arms as the air fills with crackling shards of light!', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {}

function onMagic(sourceActor, targetActor) {}
