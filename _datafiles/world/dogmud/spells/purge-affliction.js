// Purge Affliction spell script — cast/wait text in YAML; onMagic handles logic

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
