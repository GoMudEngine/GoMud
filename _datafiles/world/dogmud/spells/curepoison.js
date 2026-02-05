
// Called when the casting is initialized (cast command)
// Return false if the casting should be ignored/aborted
function onCast(sourceActor, targetActor) {

    SendUserMessage(sourceActor.UserId(), 'You begin to meditate deeply, recalling images of nature.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' enters a meditative trance.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {

    SendUserMessage(sourceActor.UserId(), 'You feel at one with the plants around you...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' sways slightly...', sourceActor.UserId());
}

// Called when the spell succeeds its cast attempt
function onMagic(sourceActor, targetActor) {

    roomId = sourceActor.GetRoomId();

    sourceUserId = sourceActor.UserId();
    sourceName = sourceActor.GetCharacterName(true);

    targetUserId = targetActor.UserId();
    targetName = targetActor.GetCharacterName(true);


    if ( sourceActor.UserId() != targetActor.UserId() ) {

        // Tell the caster about the action
        SendUserMessage(sourceUserId, 'You direct a curative energy towards '+targetName+'.');

        // Tell the room about the heal, except the source and target
        SendRoomMessage(roomId, sourceName+' directs a curative energy towards '+targetName+'.', sourceUserId, targetUserId);

        // Tell the target about the heal
        SendUserMessage(targetUserId, sourceName+' directs a curative energy towards you.');

    } else {

        // Tell the cast they did it to themselves
        SendUserMessage(sourceUserId, 'You bathe in curative energy.');

        // Tell the room about the heal, except the source and target
        SendRoomMessage(roomId, sourceName+' bathes in curative energy.', sourceUserId);

    }

    // Apply the heal to the target
    targetActor.CancelBuffWithFlag("poison");
    
}
