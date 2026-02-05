
HARM_DICE_QTY = 1;
HARM_DICE_SIDES = 6;
HARM_DICE_MOD = 2;

// Called when the casting is initialized (cast command)
// Return false if the casting should be ignored/aborted
function onCast(sourceActor, targetActor) {

    SendUserMessage(sourceActor.UserId(), 'You begin to chant softly.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' begins to chant softly.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {

    SendUserMessage(sourceActor.UserId(), 'You continue chanting, as a swirling light gathers...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' continues chanting, as a swirling light gathers...', sourceActor.UserId());
}

// Called when the spell succeeds its cast attempt
function onMagic(sourceActor, targetActor) {

    roomId = sourceActor.GetRoomId();

    harmAmt = UtilDiceRoll(HARM_DICE_QTY, HARM_DICE_SIDES) + HARM_DICE_MOD;
    harmAmtStr = String(harmAmt);

    sourceUserId = sourceActor.UserId();
    sourceName = sourceActor.GetCharacterName(true);

    targetUserId = targetActor.UserId();
    targetName = targetActor.GetCharacterName(true);

    // Tell the caster about the action
    SendUserMessage(sourceUserId, 'You let loose a magical projectile at '+targetName+', doing <ansi fg="damage">'+harmAmtStr+' hitpoints</ansi> of damage!');

    // Tell the room about the heal, except the source and target
    SendRoomMessage(roomId, sourceName+' lets loose a magical projectile at '+targetName+' hurting them!', sourceUserId, targetUserId);

    // Tell the target about the heal
    SendUserMessage(targetUserId, sourceName+' lets loose a magical projectile at you, doing <ansi fg="damage">'+harmAmtStr+' hitpoints</ansi> of damage!');

    // Apply the heal to the target
    targetActor.AddHealth(harmAmt * -1);
    
}

