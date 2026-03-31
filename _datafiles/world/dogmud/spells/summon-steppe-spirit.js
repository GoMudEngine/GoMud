// Summon Steppe Spirit — spirit wolf companion spell
// Requires: Spirit Fetish (item 40031) consumed on cast
// Spawns: Steppe Spirit Wolf (mob 243), charmed permanently
// Limit: one spirit wolf per caster

var COMPONENT_ITEM_ID = 40031;
var SUMMON_MOB_ID = 243;

function onCast(sourceActor, targetActor) {
    // Check if caster already has an active charmed mob
    if (sourceActor.HasCharmedMobs()) {
        SendUserMessage(sourceActor.UserId(), 'A steppe spirit already walks beside you. The wind carries only one voice at a time.');
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

    // Spawn the spirit wolf
    var room = GetRoom(sourceActor.GetRoomId());
    var wolf = room.SpawnMob(SUMMON_MOB_ID);
    if (!wolf) {
        SendUserMessage(sourceActor.UserId(), 'The spirits stir but refuse to take form. Something is wrong.');
        return;
    }

    // Charm permanently (99999 rounds)
    wolf.CharmSet(sourceActor.UserId(), 99999);

    SendUserMessage(sourceActor.UserId(), 'The fetish dissolves into motes of pale light. A spectral wolf coalesces from steppe wind and moonlight, its eyes burning with cold intelligence. It regards you steadily, then falls into step beside you.');
    SendRoomMessage(sourceActor.GetRoomId(), 'A ghostly wolf materializes from swirling wind and pale light, falling into step beside '+sourceActor.GetCharacterName(true)+'.', sourceActor.UserId());
}
