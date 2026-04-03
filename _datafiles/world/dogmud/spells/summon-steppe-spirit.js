// Summon Steppe Spirit — spirit wolf companion spell
// Requires: Spirit Fetish (item 40031) consumed on cast
// Spawns: Steppe Spirit Wolf (mob 243), charmed permanently
// School: manifestation — scales with charisma + manifestation skill
// Quest-gated: only castable after completing Quest 12

var COMPONENT_ITEM_ID = 40031;
var SUMMON_MOB_ID = 243;
var BASE_STAT_POOL = 120;

function onCast(sourceActor, targetActor) {
    // Check companion cap
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. Release one before calling another.');
        return false;
    }

    // Check for component
    if (!sourceActor.HasItemId(COMPONENT_ITEM_ID)) {
        SendUserMessage(sourceActor.UserId(), 'You need a spirit fetish to call the steppe spirits.');
        return false;
    }

    SendUserMessage(sourceActor.UserId(), 'You hold the spirit fetish aloft. The steppe wind rises, carrying the scent of sage and old bones...');
    SendRoomMessage(sourceActor.GetRoomId(), sourceActor.GetCharacterName(true)+' raises a small fetish overhead. The wind sharpens and the air grows cold.', sourceActor.UserId());
    return true;
}

function onWait(sourceActor, targetActor) {
    SendUserMessage(sourceActor.UserId(), 'The fetish thrums in your hand. Ghostly howling builds from nowhere and everywhere at once...');
}

function onMagic(sourceActor, targetActor) {
    // Re-check companion cap (state may have changed between cast and resolve)
    if (sourceActor.GetCompanionCount() >= sourceActor.GetMaxCompanionCount()) {
        SendUserMessage(sourceActor.UserId(), 'You have reached your limit of bound companions. The spirit cannot answer your call.');
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
        SendUserMessage(sourceActor.UserId(), 'The spirit fetish crumbles to nothing. The spell fizzles.');
        return;
    }

    // Calculate scaled stat pool: charisma and manifestation skill both contribute
    var charisma = sourceActor.GetStat('charisma');
    var manifestSkill = sourceActor.GetSkillLevel('manifestation');
    var scale = 1.0 + charisma / 200.0 + manifestSkill * 0.02;
    var scaledPool = Math.round(BASE_STAT_POOL * scale);

    // Spawn the spirit wolf with scaled stats
    var room = GetRoom(sourceActor.GetRoomId());
    var wolf = room.SpawnMobScaled(SUMMON_MOB_ID, scaledPool);
    if (!wolf) {
        SendUserMessage(sourceActor.UserId(), 'The spirits stir but refuse to take form. Something is wrong.');
        return;
    }

    // Charm permanently (99999 rounds)
    wolf.CharmSet(sourceActor.UserId(), 99999);

    // Register as companion
    sourceActor.AddCompanion(wolf.InstanceId(), 'summoned', 'Steppe Spirit Wolf');

    SendUserMessage(sourceActor.UserId(), 'The fetish dissolves into motes of pale light. A spectral wolf coalesces from steppe wind and moonlight, its eyes burning with cold intelligence. It regards you steadily, then falls into step beside you.');
    SendRoomMessage(sourceActor.GetRoomId(), 'A ghostly wolf materializes from swirling wind and pale light, falling into step beside '+sourceActor.GetCharacterName(true)+'.', sourceActor.UserId());
}
