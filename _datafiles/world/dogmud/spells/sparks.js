
DMG_DICE_QTY = 1;
DMG_DICE_SIDES = 3;

// Called when the casting is initialized (cast command)
// Return false if the casting should be ignored/aborted
function onCast(sourceActor, targetActors) {

    SendUserMessage(sourceActor.UserId(), 'You begin to chant softly.');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' begins to chant softly.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActors) {

    SendUserMessage(sourceActor.UserId(), 'You continue chanting...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' continues chanting...', sourceActor.UserId());
}

// Called when the spell succeeds its cast attempt
function onMagic(sourceActor, targetActors) {

    roomId = sourceActor.GetRoomId();

    sourceUserId = sourceActor.UserId();
    sourceName = sourceActor.GetCharacterName(true);

    for (var i = 0; i < targetActors.length; i++) {
        
        dmgAmt = UtilDiceRoll(DMG_DICE_QTY, DMG_DICE_SIDES) + 1;
        dmgAmtStr = String(dmgAmt);

        targetUserId = targetActors[i].UserId();
        targetName = targetActors[i].GetCharacterName(true);

        if ( sourceActor.UserId() != targetActors[i].UserId() ) {

            // Tell the caster about the action
            SendUserMessage(sourceUserId, 'You let loose a shower of sparks that hit '+targetName+', doing <ansi fg="damage">'+dmgAmtStr+' damage</ansi>.');

            // Tell the room about the dmg, except the source and target
            SendRoomMessage(roomId, sourceName+' stops chanting and lets loose a shower of sparks, hitting '+targetName+'.', sourceUserId, targetUserId);

            // Tell the target about the dmg
            SendUserMessage(targetUserId, sourceName+' stops chanting fires a shower of sparks at you, hitting for <ansi fg="damage">'+dmgAmtStr+' damage</ansi>.');

        } else {

            // Tell the cast they did it to themselves
            SendUserMessage(sourceUserId, 'You stop chanting and fires a shower of sparks at yourself, doing <ansi fg="damage">'+dmgAmtStr+' damage</ansi>.');

            // Tell the room about the dmg, except the source and target
            SendRoomMessage(roomId, sourceName+' stops chanting and fires a shower of sparks at themselves, hurting themselves.', sourceUserId, targetUserId);

        }

        // Apply the dmg to the target
        targetActors[i].AddHealth(dmgAmt * -1);
    }
    
}
