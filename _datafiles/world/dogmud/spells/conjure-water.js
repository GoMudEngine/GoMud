// Conjure Water Elemental
// No component required. Scales with charisma + manifestation skill.
// Spawns: Water Elemental (mob 310), charmed permanently.

var SUMMON_MOB_ID = 310;
var BASE_STAT_POOL = 80;

function onCast(sourceActor, targetActor) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. Release one before calling another.');
        return false;
    }

    SendUserMessage(sourceActor.UserId(), 'You reach into the elemental plane, feeling the cold rush of deep water answer your call...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' gestures outward, and the air grows heavy with the smell of rain and deep wells.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'The moisture in the air thickens. Water begins to spiral inward from every direction...');
}

function onMagic(sourceActor, targetActor) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The elemental cannot coalesce.');
        return;
    }

    var charisma = sourceActor.GetStat('charisma');
    var manifestSkill = sourceActor.GetSkillLevel('manifestation');
    var scale = 1.0 + charisma / 500.0 + manifestSkill * 0.02;
    var scaledPool = Math.round(BASE_STAT_POOL * scale);

    var room = GetRoom(sourceActor.GetRoomId());
    var elemental = room.SpawnMobScaled(SUMMON_MOB_ID, scaledPool);
    if (!elemental) {
        SendUserMessage(sourceActor.UserId(), 'The water scatters and soaks into the ground. Something is wrong.');
        return;
    }

    elemental.CharmSet(sourceActor.UserId(), 99999);
    sourceActor.AddCompanion(elemental.InstanceId(), 'conjured', 'Water Elemental');

    SendUserMessage(sourceActor.UserId(), 'A churning column of animated water tears through the air and crashes into form beside you, its currents spiraling with patient, cold intelligence. It regards you and waits.');
    SendRoomMessage(sourceActor.GetRoomId(), 'Water erupts from nothing and coalesces into a looming bipedal form beside '+sourceActor.GetCharacterName(true)+'.', sourceActor.UserId());
}
