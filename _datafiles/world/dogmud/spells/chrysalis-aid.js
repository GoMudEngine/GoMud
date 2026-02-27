
PLUS_SIGN_LEFT = "<ansi fg=\"green-bold\">+++</ansi> ";
PLUS_SIGN_RIGHT = " <ansi fg=\"green-bold\">+++</ansi>";

// Called when the casting is initialized (cast command)
// Return false if the casting should be ignored/aborted
function onCast(sourceActor, targetActor) {

    roomId = sourceActor.GetRoomId();

    sourceUserId = sourceActor.UserId();
    sourceName = sourceActor.GetCharacterName(true);

    targetUserId = targetActor.UserId();
    targetName = targetActor.GetCharacterName(true);

    targetHealth = targetActor.GetHealth();

    if ( targetHealth > 0 ) {
        SendUserMessage(sourceUserId, targetName+' is not in need of aid.');
        return false;
    }

    SendUserMessage(sourceUserId, PLUS_SIGN_LEFT+"You prepare to provide aid to "+targetName+"."+PLUS_SIGN_RIGHT);
    SendUserMessage(targetUserId, PLUS_SIGN_LEFT+sourceName+" prepares to apply first aid on you."+PLUS_SIGN_RIGHT);
    SendRoomMessage(roomId, PLUS_SIGN_LEFT+sourceName+" prepares to provide aid to "+targetName+"."+PLUS_SIGN_RIGHT, sourceUserId, targetUserId);

    return true;
}

function onWait(sourceActor, targetActor) {

    roomId = sourceActor.GetRoomId();

    sourceUserId = sourceActor.UserId();
    sourceName = sourceActor.GetCharacterName(true);

    targetUserId = targetActor.UserId();
    targetName = targetActor.GetCharacterName(true);

    targetHealth = targetActor.GetHealth();

    if ( targetHealth> 0 ) {
        SendUserMessage(sourceUserId, targetName+' is no longer in need of aid.');
        return false;
    }

    SendUserMessage(sourceUserId, PLUS_SIGN_LEFT+"You continue providing aid to "+targetName+"."+PLUS_SIGN_RIGHT);
    SendUserMessage(targetUserId, PLUS_SIGN_LEFT+sourceName+" continues providing aid to you."+PLUS_SIGN_RIGHT);
    SendRoomMessage(roomId, PLUS_SIGN_LEFT+sourceName+" is providing aid to "+targetName+ "."+PLUS_SIGN_RIGHT, sourceUserId, targetUserId);
}

// Called when the spell succeeds its cast attempt
function onMagic(sourceActor, targetActor) {

    roomId = sourceActor.GetRoomId();

    sourceUserId = sourceActor.UserId();
    sourceName = sourceActor.GetCharacterName(true);

    targetUserId = targetActor.UserId();
    targetName = targetActor.GetCharacterName(true);

    targetHealth = targetActor.GetHealth();
    if ( targetHealth > 0 ) {
        SendUserMessage(sourceUserId, targetName+' is no longer in need of aid.');
        return false;
    }


    // Apply the heal to the target
    targetActor.AddHealth( (targetHealth*-1) + 1 );

    SendUserMessage(sourceUserId, PLUS_SIGN_LEFT+"You stop the bleeding for "+targetName+"."+PLUS_SIGN_RIGHT);
    SendUserMessage(targetUserId, PLUS_SIGN_LEFT+sourceName+" stops your bleeding."+PLUS_SIGN_RIGHT);
    SendRoomMessage(roomId, PLUS_SIGN_LEFT+sourceName+" stops "+targetName+ " from bleeding out."+PLUS_SIGN_RIGHT, sourceUserId, targetUserId);
}
