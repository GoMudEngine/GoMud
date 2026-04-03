// Conjure Magma Elemental
// No component required. Scales with charisma + manifestation skill.
// Spawns: Magma Elemental (mob 314), charmed permanently.
// Note: magma elemental has naturalbash + return_damage — devastating in melee.

var SUMMON_MOB_ID = 314;
var BASE_STAT_POOL = 130;

function onCast(sourceActor, targetActor) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. Release one before calling another.');
        return false;
    }

    SendUserMessage(sourceActor.UserId(), 'You force your will through both the earth and fire planes simultaneously, bracing against the dual weight of what answers...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' goes rigid, face contorted, as the ground cracks and the air shimmers with sudden savage heat.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'The floor heaves. Orange light bleeds through the cracks. Something immense and blazing hot is forcing its way through...');
}

function onMagic(sourceActor, targetActor) {
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The magma elemental ruptures back to its plane.');
        return;
    }

    var charisma = sourceActor.GetStat('charisma');
    var manifestSkill = sourceActor.GetSkillLevel('manifestation');
    var scale = 1.0 + charisma / 500.0 + manifestSkill * 0.02;
    var scaledPool = Math.round(BASE_STAT_POOL * scale);

    var room = GetRoom(sourceActor.GetRoomId());
    var elemental = room.SpawnMobScaled(SUMMON_MOB_ID, scaledPool);
    if (!elemental) {
        SendUserMessage(sourceActor.UserId(), 'The ground seals shut. The magma elemental was too powerful to hold — it tore free and retreated.');
        return;
    }

    elemental.CharmSet(sourceActor.UserId(), 99999);
    sourceActor.AddCompanion(elemental.InstanceId(), 'conjured', 'Magma Elemental');

    SendUserMessage(sourceActor.UserId(), 'The floor ruptures. A towering figure of black basalt and churning lava hauls itself into the world with a sound like tectonic plates grinding. Veins of orange fire seam its shell. The heat radiating from it is enough to make the air shimmer in every direction. It stands, and the ground blackens beneath it.');
    SendRoomMessage(sourceActor.GetRoomId(), 'The floor tears apart as a massive magma elemental forces its way into the world beside '+sourceActor.GetCharacterName(true)+'. The room fills with volcanic heat.', sourceActor.UserId());
}
