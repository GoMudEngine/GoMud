// West Meadow - Room 106
// Wilderness Guide Fen teaches foraging, tracking, and stealth.
// Sub-steps: wilderness_arrive -> wilderness_forage -> wilderness_track

const guideMobId = 54;

var forageAttempted = false;
var forageTicks = 0;
var trackAttempted = false;
var trackTicks = 0;

function onEnter(user, room) {

    // Only trigger if player has done alchemy_craft but not yet wilderness_arrive
    if ( !user.HasQuest("1-alchemy_craft") || user.HasQuest("1-wilderness_arrive") ) {
        return true;
    }

    var guide = room.GetMob(guideMobId, true);
    if ( guide == null ) {
        return true;
    }

    guide.Command('emote straightens up from examining a patch of scrub.', 1.0);
    guide.Command('say Survival in the open country is about reading what is already there.', 2.0);
    guide.Command('say You do not fight the desert. You learn its logic. Water collects where the basalt dips. Shelter is where the largest outcrops cluster.', 3.5);
    guide.Command('say The grey-green brush here -- desert sage. Edible, medicinal if prepared correctly.', 5.0);
    guide.Command('say Try it now. Type <ansi fg="command">forage</ansi> and the land will give you what it has.', 6.5);

    user.GiveQuest("1-wilderness_arrive");

    return true;
}

function onCommand(cmd, rest, user, room) {

    // Detect forage command
    if ( cmd == "forage" ) {
        if ( user.HasQuest("1-wilderness_arrive") && !user.HasQuest("1-wilderness_forage") ) {
            forageAttempted = true;
            forageTicks = 0;
        }
    }

    // Detect track command
    if ( cmd == "track" ) {
        if ( user.HasQuest("1-wilderness_forage") && !user.HasQuest("1-wilderness_track") ) {
            trackAttempted = true;
            trackTicks = 0;
        }
    }

    if ( cmd == "talk" ) {
        var guide = room.GetMob(guideMobId, true);
        if ( guide != null ) {
            if ( user.HasQuest("1-wilderness_forage") && !user.HasQuest("1-wilderness_track") ) {
                guide.Command('say Something passed through here. Type <ansi fg="command">track</ansi> to read it.');
            } else if ( user.HasQuest("1-wilderness_arrive") && !user.HasQuest("1-wilderness_forage") ) {
                guide.Command('say The land has something for you. Type <ansi fg="command">forage</ansi>.');
            } else if ( user.HasQuest("1-wilderness_track") ) {
                guide.Command('say The Elder is waiting at the Observatory. North through the Alchemist, then up.');
            } else {
                guide.Command('say Come back when the Alchemist has spoken with you.');
            }
        }
        return false;
    }
    return false;
}

function onIdle(room) {

    // Handle forage detection
    if ( forageAttempted ) {
        forageTicks++;
        if ( forageTicks >= 2 ) {
            forageAttempted = false;
            forageTicks = 0;

            var players = room.GetPlayers();
            if ( players != null ) {
                for ( var i = 0; i < players.length; i++ ) {
                    var player = players[i];
                    if ( player.HasQuest("1-wilderness_arrive") && !player.HasQuest("1-wilderness_forage") ) {
                        var guide = room.GetMob(guideMobId, true);
                        if ( guide != null ) {
                            guide.Command('emote nods once.', 0.5);
                            guide.Command('say Good. Not always much -- but something.', 1.5);
                            guide.Command('say Tracking is the same principle. Something passed here. The ground tells you when, how heavy, and how fast, if you know the language.', 3.0);
                            guide.Command('say Type <ansi fg="command">track</ansi> to read it.', 4.5);
                        }
                        player.GiveQuest("1-wilderness_forage");
                    }
                }
            }
        }
    }

    // Handle track detection
    if ( trackAttempted ) {
        trackTicks++;
        if ( trackTicks >= 2 ) {
            trackAttempted = false;
            trackTicks = 0;

            var players = room.GetPlayers();
            if ( players != null ) {
                for ( var i = 0; i < players.length; i++ ) {
                    var player = players[i];
                    if ( player.HasQuest("1-wilderness_forage") && !player.HasQuest("1-wilderness_track") ) {
                        var guide = room.GetMob(guideMobId, true);
                        if ( guide != null ) {
                            guide.Command('emote studies you for a moment, then nods.', 0.5);
                            guide.Command('say You are reading it. That is the start.', 1.5);
                            guide.Command('say One more thing. In the open, being seen is a choice. Type <ansi fg="command">sneak</ansi> before you move and you become harder to notice.', 3.0);
                            guide.Command('say The Elder is above the Alchemist, at the Observatory. Go.', 4.5);
                        }
                        player.GiveQuest("1-wilderness_track");
                    }
                }
            }
        }
    }
}

function onExit(user, room) {
}

function onLoad(room) {
}
