// Conjure Fire Elemental
// No component required. Scales with charisma + manifestation skill.
// Spawns: Fire Elemental (mob 313), charmed permanently.
// Note: fire elemental has return_damage — melee attackers take burn damage.

var SUMMON_MOB_ID = 313;
var BASE_STAT_POOL = 85;

function onCast(sourceActor, targetActor) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. Release one before calling another.');
        return false;
    }

    return true;
}

function onMagic(sourceActor, targetActor) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The elemental blazes free and vanishes.');
        return;
    }

    var charisma = sourceActor.GetStat('charisma');
    var manifestSkill = sourceActor.GetSkillLevel('manifestation');
    var scale = 1.0 + charisma / 500.0 + manifestSkill * 0.02;
    var scaledPool = Math.round(BASE_STAT_POOL * scale);

    var room = GetRoom(sourceActor.GetRoomId());
    var elemental = room.SpawnMobScaled(SUMMON_MOB_ID, scaledPool);
    if (!elemental) {
        SendUserMessage(sourceActor.UserId(), 'The flame erupts and extinguishes in an instant. The elemental tore free of the binding.');
        return;
    }

    elemental.CharmSet(sourceActor.UserId(), 99999);
    sourceActor.AddCompanion(elemental.InstanceId(), 'conjured', 'Fire Elemental');

    SendUserMessage(sourceActor.UserId(), 'A column of living flame tears through into the world, burning deep orange at the core and blue-white at its edges. The heat is blistering at three paces. It crackles once — furious, shackled — then turns that fury outward, awaiting your direction.');
    SendRoomMessage(sourceActor.GetRoomId(), 'The air ignites in a pillar of furious flame beside '+sourceActor.GetCharacterName(true)+'. Scorched footprints glow briefly on the floor.', sourceActor.UserId());
}
