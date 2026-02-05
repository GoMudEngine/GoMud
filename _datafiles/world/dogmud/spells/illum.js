
// Called when the casting is initialized (cast command)
// Return false if the casting should be ignored/aborted
function onCast(sourceActor, targetActor) {

    SendUserMessage(sourceActor.UserId(), 'You begin to chant softly.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' begins to chant softly.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {

    SendUserMessage(sourceActor.UserId(), 'You gather threads of light...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' is gathering threads of light...', sourceActor.UserId());
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
        SendUserMessage(sourceUserId, 'You materialize a glowing orb.');

        // Tell the room about the heal, except the source and target
        SendRoomMessage(roomId, sourceName+' materializes a glowing orb, which follows '+targetName+' around.', sourceUserId, targetUserId);

        // Tell the target about the heal
        SendUserMessage(targetUserId, sourceName+' materializes a glowing orb, which follows you around.');

    } else {

        // Tell the cast they did it to themselves
        SendUserMessage(sourceUserId, 'You materialize a glowing orb.');

        // Tell the room about the heal, except the source and target
        SendRoomMessage(roomId, sourceName+' materializes a glowing orb, which follows them around.', sourceUserId);

    }

    // Apply the heal to the target
    targetActor.GiveBuff(1, "spell");
    
}
