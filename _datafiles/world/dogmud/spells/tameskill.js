
const UnlimitedMinutes = -1;
const SixtyMinutes = 60*60;
const FifteenMinutes = 60*15;
const FiveMinutes = 60*5;


// Called when the casting is initialized (cast command)
// Return false if the casting should be ignored/aborted
function onCast(sourceActor, targetActor) {

    if ( !targetActor.IsTameable() ) {
        SendUserMessage(sourceActor.UserId(), targetActor.GetCharacterName(true)+' can\'t be tamed!');
        return false;
    }

    if ( targetActor.IsCharmed() ) {
        SendUserMessage(sourceActor.UserId(), 'Already friendly!');
        return false;
    }

    skillLevel = sourceActor.GetSkillLevel("tame");
    charmCt = sourceActor.GetCharmCount();
    if ( charmCt >= skillLevel+1 ) {
        SendUserMessage(sourceActor.UserId(), 'You can only have '+String(skillLevel+1)+' creatures following you at a time.');
        return true;    
    }

    allTameSkills = sourceActor.GetTameMastery();
    proficiencyModifier = allTameSkills[targetActor.MobTypeId()];
    if ( proficiencyModifier == null ) {
        SendUserMessage(sourceActor.UserId(), 'You don\'t know how to tame a '+targetActor.GetCharacterName(true)+'.');
        return true;
    }

    if ( sourceActor.GetCharmCount() >= sourceActor.GetMaxCharmCount() ) {
        sourceActor.SendText(`You already have too many followers.`);
    }
    
    chance = sourceActor.GetChanceToTame(targetActor)+sourceActor.GetStatMod(`tame`);
    
    SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You have a <ansi fg="151">'+chance+'% chance</ansi> to successfully tame the '+targetActor.GetCharacterName(true)+'.</ansi>');
    SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You begin to dance in front of the '+targetActor.GetCharacterName(true)+'.</ansi>');
    SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' performs a carefully choreographed dance in front of '+targetActor.GetCharacterName(true)+'.</ansi>', sourceActor.UserId());

    return true;
}

function onWait(sourceActor, targetActor) {

    switch ( UtilDiceRoll(1, 11) ) {
        case 1:
            SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You make a series of gutteral sounds that seem to distract the '+targetActor.GetCharacterName(true)+'.</ansi>');
            SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' makes a series of gutteral sounds that seem to distract the '+targetActor.GetCharacterName(true)+'.</ansi>', sourceActor.UserId());
            break;
        case 2:
            SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You continue to chant in front of the '+targetActor.GetCharacterName(true)+'.</ansi>');
            SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' continues to chant in front of '+targetActor.GetCharacterName(true)+'.</ansi>', sourceActor.UserId());
            break;
        case 3:
            SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You fall to the floor and slither like a snake in front of the '+targetActor.GetCharacterName(true)+'.</ansi>');
            SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' falls to the floor and slithers like a snake in front of the '+targetActor.GetCharacterName(true)+'.</ansi>', sourceActor.UserId());
            break;
        case 4:
            SendUserMessage(sourceActor.UserId(), '<ansi fg="219">Your body stiffens, and the '+targetActor.GetCharacterName(true)+' becomes alert.</ansi>');
            SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' suddenly stiffens, and the '+targetActor.GetCharacterName(true)+' becomes alert.</ansi>', sourceActor.UserId());
            break;
        case 5:
            SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You run in a circle around the '+targetActor.GetCharacterName(true)+'.</ansi>');
            SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' runs in a circle around the '+targetActor.GetCharacterName(true)+'.</ansi>', sourceActor.UserId());
            break;
        case 6:
            SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You purr ever so gently.</ansi>');
            SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' purrs ever so gently.</ansi>', sourceActor.UserId());
            break;
        case 7:
            SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You shake your fist angrily.</ansi>');
            SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' shakes their fist angrily.</ansi>', sourceActor.UserId());
            break;
        case 8:
            SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You jingle a little bell.</ansi>');
            SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' jingles a little bell.</ansi>', sourceActor.UserId());
            break;
        case 9:
            SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You raise one eyebrow... then the other!</ansi>');
            SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' raises one eyebrow... then the other!</ansi>', sourceActor.UserId());
            break;
        case 10:
            SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You slowly raise your hands upwards, and then CLAP them together loudly!</ansi>');
            SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' slowly raises their hands upwards, and then CLAPS them together loudly!</ansi>', sourceActor.UserId());
            break;
        default:
            SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You whistle several times, changing your pitch ever so slightly.</ansi>');
            SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceActor.GetCharacterName(true)+' whistles several times, changing your pitch ever so slightly.</ansi>', sourceActor.UserId());
    }

}

// Called when the spell succeeds its cast attempt
// Return true to ignore any auto-retaliation from the target
function onMagic(sourceActor, targetActor) {

    if ( targetActor.IsCharmed() ) {
        SendUserMessage(sourceActor.UserId(), 'Already friendly!');
        return false;
    }

    if ( sourceActor.GetCharmCount() >= sourceActor.GetMaxCharmCount() ) {
        sourceActor.SendText(`You already have too many followers.`);
    }

    targetName = targetActor.GetCharacterName(true);
    sourceName = sourceActor.GetCharacterName(true);

    successChance = sourceActor.GetChanceToTame(targetActor)+sourceActor.GetStatMod(`tame`);

    randNumber = UtilDiceRoll(1, 100) - 1;
    
    if ( randNumber >= successChance ) {
        SendUserMessage(sourceActor.UserId(), '<ansi fg="219">The '+targetName+' <ansi fg="182">RESISTS</ansi> your attempt to tame it!</ansi>');
        SendRoomMessage(sourceActor.GetRoomId(), '<ansi fg="219">The '+targetName+' <ansi fg="182">RESISTS</ansi> '+sourceName+'\'s attempt to tame it!</ansi>', sourceActor.UserId());
        
        targetActor.Command(`attack ` + sourceActor.ShorthandId());
        return false;
    }

    SendUserMessage(sourceActor.UserId(), '<ansi fg="219">You <ansi fg="151">SUCCESSFULLY</ansi> tame the '+targetName+'!</ansi>');
    SendRoomMessage(sourceActor.GetRoomId(), `<ansi fg="219">`+sourceName+' <ansi fg="151">SUCCESSFULLY</ansi> tames the '+targetName+'!</ansi>', sourceActor.UserId());
    
    skillLevel = sourceActor.GetSkillLevel("tame");
    tameRounds = 0;
    switch( skillLevel ) {
        case 4:
            tameRounds = UnlimitedMinutes;
            break;
        case 3:
            tameRounds = UtilGetSecondsToRounds(SixtyMinutes);
            break;
        case 2:
            tameRounds = UtilGetSecondsToRounds(SixtyMinutes);
            break;
        default:
            tameRounds = UtilGetSecondsToRounds(SixtyMinutes);
    }

    targetActor.CharmSet(sourceActor.UserId(), tameRounds, "emote reverts to a wild state.");

    // Tell the caster about the action
    if ( tameRounds == UnlimitedMinutes ) {
        SendUserMessage(sourceActor.UserId(), 'The '+targetName+' has been tamed by you!');
    } else {
        SendUserMessage(sourceActor.UserId(), 'The '+targetName+' has been tamed by you for '+String(tameRounds)+' rounds!');
    }

    // Tell the room about the heal, except the source and target
    SendRoomMessage(sourceActor.GetRoomId(), sourceName+' tames the '+targetName+'!', sourceActor.UserId(), targetActor.UserId());

 
    return true;
}
