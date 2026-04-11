// Summon Hive Swarm — permanent swarm companion spell
// Requires: Hive Fragment (item 40011) consumed on cast
// Spawns: Hive Swarm (mob 111), charmed permanently
// School: manifestation — scales with charisma + manifestation skill

var COMPONENT_ITEM_ID = 40011;
var SUMMON_MOB_ID = 111;
var BASE_STAT_POOL = 18;

function onCast(sourceActor, targetActor) {
    // Check companion cap
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. Release one before calling another.');
        return false;
    }

    // Check for component
    if (!sourceActor.HasItemId(COMPONENT_ITEM_ID)) {
        SendUserMessage(sourceActor.UserId(), 'You need a Hive Fragment to call forth the swarm.');
        return false;
    }

    return true;
}

function onMagic(sourceActor, targetActor) {
    // Re-check companion cap (state may have changed between cast and resolve)
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The swarm cannot coalesce.');
        return;
    }

    // Consume the component
    var items = sourceActor.GetBackpackItems();
    var consumed = false;
    for (var i = 0; i < items.length; i++) {
        if (items[i].ItemId() === COMPONENT_ITEM_ID) {
            sourceActor.TakeItem(items[i]);
            consumed = true;
            break;
        }
    }
    if (!consumed) {
        SendUserMessage(sourceActor.UserId(), 'The Hive Fragment is gone. The spell fizzles.');
        return;
    }

    // Calculate scaled stat pool: charisma and manifestation skill both contribute
    var charisma = sourceActor.GetStat('charisma');
    var manifestSkill = sourceActor.GetSkillLevel('manifestation');
    var scale = 1.0 + charisma / 200.0 + manifestSkill * 0.02;
    var scaledPool = Math.round(BASE_STAT_POOL * scale);

    // Spawn the swarm with scaled stats
    var room = GetRoom(sourceActor.GetRoomId());
    var swarm = room.SpawnMobScaled(SUMMON_MOB_ID, scaledPool);
    if (!swarm) {
        SendUserMessage(sourceActor.UserId(), 'The tiny organisms scatter and dissolve. Something is wrong.');
        return;
    }

    // Charm permanently (99999 rounds)
    swarm.CharmSet(sourceActor.UserId(), 99999);

    // Register as companion
    sourceActor.AddCompanion(swarm.InstanceId(), 'summoned', 'Hive Swarm');

    SendUserMessage(sourceActor.UserId(), 'The fragment erupts into a roiling cloud of iridescent organisms — a Hive Swarm coalesces around you, awaiting your command!');
    SendRoomMessage(sourceActor.GetRoomId(), 'A chitinous fragment shatters in '+sourceActor.GetCharacterName(true)+"'s hand, releasing a dense swarm of tiny Chrysalis creatures that swirl into formation.", sourceActor.UserId());
}
