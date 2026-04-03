// Conjure Air Elemental
// No component required. Scales with charisma + manifestation skill.
// Spawns: Air Elemental (mob 312), charmed permanently.

var SUMMON_MOB_ID = 312;
var BASE_STAT_POOL = 70;

function onCast(sourceActor, targetActor) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. Release one before calling another.');
        return false;
    }

    SendUserMessage(sourceActor.UserId(), 'You tear an opening in the veil and call to the shrieking currents beyond...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+"'s hair and clothing snap sideways as if caught in sudden wind.", sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'A high, keening sound builds from nowhere. The air pressure drops sharply as something tears through...');
}

function onMagic(sourceActor, targetActor) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The elemental scatters on the wind.');
        return;
    }

    var charisma = sourceActor.GetStat('charisma');
    var manifestSkill = sourceActor.GetSkillLevel('manifestation');
    var scale = 1.0 + charisma / 500.0 + manifestSkill * 0.02;
    var scaledPool = Math.round(BASE_STAT_POOL * scale);

    var room = GetRoom(sourceActor.GetRoomId());
    var elemental = room.SpawnMobScaled(SUMMON_MOB_ID, scaledPool);
    if (!elemental) {
        SendUserMessage(sourceActor.UserId(), 'The wind screams and disperses. The elemental refuses the binding.');
        return;
    }

    elemental.CharmSet(sourceActor.UserId(), 99999);
    sourceActor.AddCompanion(elemental.InstanceId(), 'conjured', 'Air Elemental');

    SendUserMessage(sourceActor.UserId(), 'A column of howling wind tears into existence beside you, lightning threading its core in silent white arcs. Papers scatter. Flames gutter. The air elemental coils once, then holds still — barely.');
    SendRoomMessage(sourceActor.GetRoomId(), 'A screaming vortex of wind explodes into being beside '+sourceActor.GetCharacterName(true)+', lightning crackling through its core.', sourceActor.UserId());
}
