// Conjure Earth Elemental
// No component required. Scales with charisma + manifestation skill.
// Spawns: Earth Elemental (mob 311), charmed permanently.

var SUMMON_MOB_ID = 311;
var BASE_STAT_POOL = 90;

function onCast(sourceActor, targetActor) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. Release one before calling another.');
        return false;
    }

    return true;
}

function onMagic(sourceActor, targetActor) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The elemental cannot rise.');
        return;
    }

    var charisma = sourceActor.GetStat('charisma');
    var manifestSkill = sourceActor.GetSkillLevel('manifestation');
    var scale = 1.0 + charisma / 500.0 + manifestSkill * 0.02;
    var scaledPool = Math.round(BASE_STAT_POOL * scale);

    var room = GetRoom(sourceActor.GetRoomId());
    var elemental = room.SpawnMobScaled(SUMMON_MOB_ID, scaledPool);
    if (!elemental) {
        SendUserMessage(sourceActor.UserId(), 'The earth settles and the trembling stops. Something resisted the call.');
        return;
    }

    elemental.CharmSet(sourceActor.UserId(), 99999);
    sourceActor.AddCompanion(elemental.InstanceId(), 'conjured', 'Earth Elemental');

    SendUserMessage(sourceActor.UserId(), 'The floor cracks apart as a massive figure of stone and compacted earth heaves itself upright. It turns two amber fissures toward you — eyes that have watched mountains form. Then it stands, waiting.');
    SendRoomMessage(sourceActor.GetRoomId(), 'The ground cracks open and a towering figure of living rock hauls itself into the world beside '+sourceActor.GetCharacterName(true)+'.', sourceActor.UserId());
}
