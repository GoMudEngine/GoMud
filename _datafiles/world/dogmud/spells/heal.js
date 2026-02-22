
HEAL_DICE_QTY = 2;
HEAL_DICE_SIDES = 3;

// Called when the casting is initialized (cast command)
// Return false if the casting should be ignored/aborted
function onCast(sourceActor, targetActor) {

    SendUserMessage(sourceActor.UserId(), 'You begin to chant softly.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' begins to chant softly.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {

    SendUserMessage(sourceActor.UserId(), 'You continue chanting...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' continues chanting...', sourceActor.UserId());
}

// Called when the spell succeeds its cast attempt
function onMagic(sourceActor, targetActor) {

    roomId = sourceActor.GetRoomId();

    healAmt = UtilDiceRoll(HEAL_DICE_QTY, HEAL_DICE_SIDES);

    sourceUserId = sourceActor.UserId();
    sourceName = sourceActor.GetCharacterName(true);

    targetUserId = targetActor.UserId();
    targetName = targetActor.GetCharacterName(true);

    if ( sourceActor.UserId() != targetActor.UserId() ) {

        // Tell the caster about the action
        SendUserMessage(sourceUserId, 'You stop chanting and touch '+targetName+' with glowing hands, healing ' + ColorWrap(UtilGetHealDescription(healAmt, targetActor.GetHealthMax()), 'healing') + '.');

        // Tell the room about the heal, except the source and target
        SendRoomMessage(roomId, sourceName+' stops chanting and touches '+targetName+' with glowing hands, providing health.', sourceUserId, targetUserId);

        // Tell the target about the heal
        SendUserMessage(targetUserId, sourceName+' stops chanting and touches you with glowing hands, healing ' + ColorWrap(UtilGetHealDescription(healAmt, targetActor.GetHealthMax()), 'healing') + '.');

    } else {

        // Tell the caster they did it to themselves
        SendUserMessage(sourceUserId, 'You stop chanting and embrace yourself with glowing hands, healing ' + ColorWrap(UtilGetHealDescription(healAmt, targetActor.GetHealthMax()), 'healing') + '.');

        // Tell the room about the heal, except the source and target
        SendRoomMessage(roomId, sourceName+' stops chanting and embraces themselves with glowing hands, providing health.', sourceUserId, targetUserId);

    }

    // Apply the heal to the target
    targetActor.AddHealth(healAmt);
    
}
