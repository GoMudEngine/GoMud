function onCommand_talk(rest, mob, room) {
    mob.Command('say I am Sable. I tend the rift archway.');
    mob.Command('say If you seek passage to dangerous places, ask me about portals.');
    return true;
}

function onAsk(mob, room, eventDetails) {
    var user = GetUser(eventDetails.sourceId);
    if (!user) return false;

    var text = (eventDetails.askText || '').toLowerCase().trim();

    // List available zones
    if (text === 'portal' || text === 'portals' || text === 'zones'
        || text === 'instance' || text === 'instances' || text === 'rift'
        || text === 'rifts') {
        mob.Command('say I can open rifts to dangerous places, for a price.');
        mob.Command('say Currently I can reach the Arena and the Planar Oasis.');
        mob.Command('say Tell me the place and how much gold you wish to invest.');
        mob.Command('say The more gold, the more dangerous -- and rewarding.');
        SendUserMessage(user.UserId(),
            '<ansi fg="181">  [Try: ask sable arena 200 or ask sable oasis 300]</ansi>');
        return true;
    }

    // Parse "<zone> <gold>" format
    var parts = text.split(' ');
    if (parts.length >= 2) {
        var zoneName = parts[0];
        var goldAmount = parseInt(parts[1], 10);

        if (isNaN(goldAmount) || goldAmount <= 0) {
            mob.Command('say Name a place and an amount of gold.');
            return true;
        }

        var zoneMap = {
            'arena': 'Instance Arena',
            'oasis': 'Instance Planar Oasis'
        };

        var templateZone = zoneMap[zoneName];
        if (!templateZone) {
            mob.Command('say I do not know of such a place.');
            return true;
        }

        if (goldAmount < 100) {
            mob.Command('say That barely covers the runes. I need at least 100 gold.');
            return true;
        }

        if (user.GetGold() < goldAmount) {
            mob.Command('say You do not have that much gold.');
            return true;
        }

        user.AddGold(-goldAmount);

        var entryRoomId = CreateInstance(templateZone, goldAmount,
            user.UserId(), room.RoomId());

        if (entryRoomId > 0) {
            var exitKey = zoneName + '-portal';
            var exitTitle = zoneName + ' portal';
            room.AddTemporaryExit(exitKey, exitTitle, entryRoomId, '30 real minutes');
            mob.Command('say The rift to the ' + zoneName + ' is open. Type ' + exitKey + ' to enter.');
            mob.Command('say It will hold for a time. Do not tarry.');
            SendRoomMessage(room.RoomId(),
                'The stone archway flares with energy. A shimmering portal appears.',
                0);
        } else {
            user.AddGold(goldAmount);
            mob.Command('say Something went wrong. The planes resist me. Your gold is returned.');
        }

        return true;
    }

    // General greetings / intro
    if (text === 'hello' || text === 'hi' || text === 'help'
        || text === 'who' || text === 'what' || text === 'quest') {
        mob.Command('say I am Sable. I open rifts to places best left undisturbed.');
        mob.Command('say For the right price in gold, I can send you and your party somewhere dangerous.');
        mob.Command('say Ask me about portals if you want to know more.');
        SendUserMessage(user.UserId(),
            '<ansi fg="181">  [Try: ask sable portals]</ansi>');
        return true;
    }

    // Single word zone queries
    if (text === 'arena') {
        mob.Command('say The Arena is a brutal proving ground. Death means expulsion.');
        mob.Command('say You cannot recall from within. Tell me how much gold to invest.');
        return true;
    }
    if (text === 'oasis') {
        mob.Command('say The Planar Oasis shimmers between worlds. Dangerous, but you may return if you fall.');
        mob.Command('say Recall magic still works there. Tell me how much gold to invest.');
        return true;
    }

    return false;
}
