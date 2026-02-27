// Purge Affliction spell script — flavor only; effects resolved in Go (Stage 11.4)

function onCast(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You focus Chrysalis energy inward, seeking out foreign agents.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' enters a meditative focus, Chrysalis energy pulsing softly.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'You feel the Chrysalis isolating the toxins...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' sways slightly as energy pulses through them...', sourceActor.UserId());
}

function onMagic(sourceActor, targetActor) {
    roomId = sourceActor.GetRoomId();
    sourceUserId = sourceActor.UserId();
    sourceName = sourceActor.GetCharacterName(true);
    targetUserId = targetActor.UserId();
    targetName = targetActor.GetCharacterName(true);

    if ( sourceActor.UserId() != targetActor.UserId() ) {
        SendUserMessage(sourceUserId, 'You direct purging energy towards '+targetName+'.');
        SendRoomMessage(roomId, sourceName+' directs purging energy towards '+targetName+'.', sourceUserId, targetUserId);
        SendUserMessage(targetUserId, sourceName+' purges the afflictions from your body.');
    } else {
        SendUserMessage(sourceUserId, 'You purge the afflictions from your body.');
        SendRoomMessage(roomId, sourceName+' purges their afflictions.', sourceUserId);
    }

    targetActor.CancelBuffWithFlag("poison");
}
