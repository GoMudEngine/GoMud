// Academy Hall - Room 113
// Chrysalis Priest delivers Awakening Rite: explains mutations, gives first mutation, starts quest.

const priestMobId = 50;

var ceremonyUnderway = false;
var ceremonyTicks    = 0;

function onEnter(user, room) {

    // Only trigger intro if player hasn't received the Awakening Rite yet
    // Check both quest progress AND whether they already have any mutations
    // (belt-and-suspenders: protects against corrupted quest state)
    if ( user.HasQuest("1-mutation") || user.GetMutationCount() > 0 ) {
        return true;
    }

    // Lock all exits for the duration of the Awakening Rite
    room.SetLocked("north", true);
    room.SetLocked("south", true);
    room.SetLocked("east", true);
    room.SetLocked("west", true);

    // If a ceremony is already underway (second player walks in), give the rite quietly and wait out the timer
    if ( ceremonyUnderway ) {
        var mutId = user.RollMutation();
        if ( mutId != "" ) { user.GiveMutation(mutId); }
        user.AddGold(10);
        user.GiveQuest("1-start");
        user.GiveQuest("1-mutation");
        return true;
    }

    var priest = room.GetMob(priestMobId, true);
    if ( priest == null ) {
        // No priest -- give the rite silently and release immediately
        var mutId = user.RollMutation();
        if ( mutId != "" ) { user.GiveMutation(mutId); }
        user.AddGold(10);
        user.GiveQuest("1-start");
        user.GiveQuest("1-mutation");
        room.SetLocked("north", false);
        room.SetLocked("south", false);
        room.SetLocked("east", false);
        room.SetLocked("west", false);
        return true;
    }

    priest.Command('say Be still for a moment.', 1.0);
    priest.Command('say Type <ansi fg="command">look</ansi> at any time to examine your surroundings. Type <ansi fg="command">help</ansi> if you are unsure what commands are available -- there is documentation for most things.', 2.5);
    priest.Command('say You are newly arrived in Gaius. You may not yet understand what that means.', 5.0);
    priest.Command('say The world here is not like any world you have come from. Gaius is made of belief -- belief shapes the land, the creatures, the rules of cause and effect.', 6.5);
    priest.Command('say And it shapes people.', 8.5);
    priest.Command('say The Chrysalis has been cataloguing that shaping for four hundred years. We know what to expect. You may not.', 10.0);
    priest.Command('say The Chrysalis is the name we give to the form between what was and what will be. Every person who has passed through this hall was once standing where you are standing.', 12.0);
    priest.Command('say In Gaius, that becoming is not a metaphor. The body remembers stress, conflict, effort. Over time, it changes. We call these changes Mutations.', 14.5);
    priest.Command('say Every Mutation is the body saying yes to something. Yes to strength, yes to perception, yes to endurance. There are costs -- the Chrysalis does not lie about that -- but a Priest does not dwell on costs. We dwell on what you are becoming.', 16.5);
    priest.Command('say The first Mutation is always a gift -- or a warning. The Chrysalis performs the Awakening Rite to initiate it before the body does something unexpected on its own.', 18.5);
    priest.Command('emote steps forward and places two fingers lightly on your forehead. The air around you hums very quietly.', 20.5);
    priest.Command('emote withdraws their hand. The hum fades.', 22.5);
    priest.Command('say It has begun. You will feel the change over time -- in combat, in effort, in moments of stress. It is part of you now.', 24.0);
    priest.Command('say You can type <ansi fg="command">mutations</ansi> at any time to see what the change has wrought.', 25.5);
    priest.Command('say Now. Before you leave this place, you have six trials to complete. Each instructor teaches a different skill.', 27.0);
    priest.Command('say First: visit the Merchant on Market Street, one step south of here. She will show you how commerce works here.', 28.5);
    priest.Command('say After that, find the Combat Trainer in the Training Yard -- go east, just beyond here. Then the Forge to the east, the Alchemist west of the well, the Wilderness Guide in the West Meadow, and finally the Elder at the Observatory above.', 30.0);
    priest.Command('say When all six are done, the Basin Warden at the south gate will let you pass. Type <ansi fg="command">quest</ansi> at any time if you lose your place.', 32.0);
    priest.Command('say There is a map of the wider world set into the floor. Type <ansi fg="command">look mosaic</ansi> to examine it before you leave.', 33.5);

    // Perform the Awakening Rite: roll and give a random mutation
    var mutId = user.RollMutation();
    if ( mutId != "" ) {
        user.GiveMutation(mutId);
    }

    // Give 10 gold as Academy stipend so the player can buy from the merchant
    user.AddGold(10);

    // Start the Sanctum Trials and mark mutation step done
    user.GiveQuest("1-start");
    user.GiveQuest("1-mutation");

    // Start ceremony timer -- exits unlock after sufficient ticks
    ceremonyUnderway = true;
    ceremonyTicks    = 0;

    return true;
}

function onCommand(cmd, rest, user, room) {

    // Block movement during the Awakening Rite -- have the Priest speak rather than a generic locked message
    if ( ceremonyUnderway ) {
        if ( cmd == "north" || cmd == "south" || cmd == "east" || cmd == "west" ||
             cmd == "n"     || cmd == "s"     || cmd == "e"    || cmd == "w"    || cmd == "go" ) {
            var priest = room.GetMob(priestMobId, true);
            if ( priest != null ) {
                priest.Command('say The Rite is not yet complete. A moment more.');
            }
            return true;
        }
    }

    // Prevent picking up the mosaic -- it is set into the floor
    if ( (cmd == "get" || cmd == "take" || cmd == "grab") && rest != null && rest != "" ) {
        var restLow = rest.toLowerCase();
        if ( restLow.indexOf("map") >= 0 || restLow.indexOf("mosaic") >= 0 ) {
            user.SendText('The mosaic is set into the floor. It is not going anywhere.');
            return true;
        }
    }

    // Map examination handler -- intercept before engine looks for item
    if ( (cmd == "look" || cmd == "examine") && rest != null && rest != "" ) {
        var target = rest.toLowerCase();
        if ( target == "map" || target == "mosaic" || target == "mosaic map" ) {
            user.SendText('');
            user.SendText('+------------------------------------------------------+');
            user.SendText('|  THE WINDWARD MARCHES                                |');
            user.SendText('|  (partial -- much of Gaius remains uncharted)        |');
            user.SendText('+------------------------------------------------------+');
            user.SendText('|                                                      |');
            user.SendText('|  [NEW PLYMOUTH *]               [? ? ?]             |');
            user.SendText('|        |         \\                                   |');
            user.SendText('|   [TIDEMARK]      \\         [STILLWATER]            |');
            user.SendText('|        |           \\              |                 |');
            user.SendText('|   [GREENFORD]   [AMBER VALLEY]   |                 |');
            user.SendText('|         \\              \\          |                 |');
            user.SendText('|          \\---------[CONFLUENCE]                    |');
            user.SendText('|                         |                           |');
            user.SendText('|                   . . . . . . .                     |');
            user.SendText('|                  .   unmapped   .                   |');
            user.SendText('|                   . . . . . . .                     |');
            user.SendText('|                    /~~~~~~~~~~~\\                    |');
            user.SendText('|                   |  SANCTUM    |                   |');
            user.SendText('|                   |   BASIN     |  <- you are here  |');
            user.SendText('|                    \\~~~~~~~~~~~/                    |');
            user.SendText('+------------------------------------------------------+');
            user.SendText('| * city  ~ plateau rim  . unmapped  \\ road           |');
            user.SendText('+------------------------------------------------------+');
            user.SendText('');
            return true;
        }
    }

    if ( cmd == "talk" || cmd == "greet" ) {
        var priest = room.GetMob(priestMobId, true);
        if ( priest != null ) {
            if ( user.HasQuest("1-shopping_arrive") ) {
                priest.Command('say Your Awakening is behind you. The trials are ahead. Type <ansi fg="command">quest</ansi> to see what remains.');
            } else if ( user.HasQuest("1-mutation") ) {
                priest.Command('say Visit the Merchant on Market Street -- east to Town Square, then south. She will show you how commerce works here.');
            } else {
                priest.Command('say Step closer. The Awakening Rite will not wait.');
            }
        }
        return false;
    }
    return false;
}

function onIdle(room) {
    if ( !ceremonyUnderway ) {
        return;
    }

    ceremonyTicks++;

    // Hold exits locked for ~6 ticks -- covers through the ritual emote and "It has begun" lines
    if ( ceremonyTicks < 6 ) {
        return;
    }

    room.SetLocked("north", false);
    room.SetLocked("south", false);
    room.SetLocked("east", false);
    room.SetLocked("west", false);

    ceremonyUnderway = false;
    ceremonyTicks    = 0;
}

function onExit(user, room) {
}

function onLoad(room) {
    // Ensure exits are open and ceremony state is clean on server restart
    ceremonyUnderway = false;
    ceremonyTicks    = 0;
    room.SetLocked("north", false);
    room.SetLocked("south", false);
    room.SetLocked("east", false);
    room.SetLocked("west", false);
}
