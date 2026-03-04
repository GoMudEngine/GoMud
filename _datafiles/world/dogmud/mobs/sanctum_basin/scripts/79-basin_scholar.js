// Basin Scholar - Mob 79
// Handles quest item delivery for Quest 3 (The Scholar's Collection).
// Either item can be delivered first -- totem or spore sac.

const BONE_TOTEM_ID = 14;
const SPORE_SAC_ID = 40008;

function onGive(mob, room, eventDetails) {

    if ( eventDetails.sourceType == "mob" ) { return false; }

    var user = GetUser(eventDetails.sourceId);
    if ( user == null ) { return false; }

    // Only care about item gives, not gold
    if ( !eventDetails.item || !eventDetails.item.ItemId ) {
        return false;
    }

    // Must have started the quest
    if ( !user.HasQuest("3-start") ) {
        mob.Command('say Interesting, but not what my research requires. Thank you though.');
        return true;
    }

    // Already done
    if ( user.HasQuest("3-end") ) {
        mob.Command('say You have already given me everything I need. Thank you, truly.');
        return true;
    }

    var itemId = eventDetails.item.ItemId;
    var hasTotem = user.HasQuest("3-totem");
    var hasSac = user.HasQuest("3-sac");

    // Bone totem delivery
    if ( itemId == BONE_TOTEM_ID ) {
        if ( hasTotem ) {
            mob.Command('say I already have a totem specimen. Thank you though.');
            return true;
        }
        if ( hasSac ) {
            // Sac was first, totem completes the quest
            user.GiveQuest("3-end");
            mob.Command('say Exquisite! The carving technique is post-mutation -- look at how the finger grooves accommodate clawed digits.', 1.0);
            mob.Command('say That completes the set. You have given the guild more data than a decade of field expeditions. I am deeply grateful.', 3.0);
            return true;
        }
        // Totem is first
        user.GiveQuest("3-totem");
        mob.Command('say Exquisite. The carving technique is post-mutation -- look at how the finger grooves accommodate clawed digits.', 1.0);
        mob.Command('say Now I need a spore sac from the cave crawlers deeper in the tunnels. It will complete the ecological picture.', 3.0);
        return true;
    }

    // Spore sac delivery
    if ( itemId == SPORE_SAC_ID ) {
        if ( hasSac ) {
            mob.Command('say I already have a spore specimen. Thank you though.');
            return true;
        }
        if ( hasTotem ) {
            // Totem was first, sac completes the quest
            user.GiveQuest("3-end");
            mob.Command('say A spore sac! And intact! The spore density is extraordinary.', 1.0);
            mob.Command('say That completes the set. You have given the guild more data than a decade of field expeditions. I am deeply grateful.', 3.0);
            return true;
        }
        // Sac is first
        user.GiveQuest("3-sac");
        mob.Command('say A spore sac! And intact! The spore density is extraordinary. This tells us so much about the tunnel biome.', 1.0);
        mob.Command('say Now I need a bone totem -- the carved figurines from the communal shelves deeper in the warren.', 3.0);
        return true;
    }

    // Not a quest item
    mob.Command('say Interesting, but not what my research requires. Thank you though.');
    return true;
}
