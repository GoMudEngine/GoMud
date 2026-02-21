// Observatory - Room 116
// Elder Saris explains the Fold and belief-made-real, teaches throw-stone, advances magic quest.

const elderMobId = 55;

function onEnter(user, room) {

    // Only trigger if player has magic quest and hasn't completed it
    if ( !user.HasQuest("1-magic") || user.HasQuest("1-graduate") ) {
        return false;
    }

    var elder = room.GetMob(elderMobId, true);
    if ( elder == null ) {
        return false;
    }

    elder.Command('emote turns from the sighting device without haste.', 1.0);
    elder.Command('say You see three moons. What you cannot see is that each one has a different phase of influence -- gravitational, tidal, and what we call the Fold pressure.', 2.0);
    elder.Command('say The Fold is the substrate of reality on Gaius. It is what belief acts upon. When enough people believe something strongly enough, the Fold reshapes to accommodate it.', 4.0);
    elder.Command('say This is not metaphor. This is mechanics. The creatures you fought in that cave -- their existence is sustained by something believing in them, including them believing in themselves.', 6.0);
    elder.Command('say Magic is directed belief, focused through trained intent. Spells are not incantations. They are concentrated conviction applied to the Fold with precision.', 8.0);
    elder.Command('say The simplest application: kinetic force. You believe the stone will move with purpose. The Fold agrees. The stone moves with purpose.', 10.0);
    elder.Command('say I have placed that belief-pattern in your mind now. You will find you can use it.', 12.0);
    elder.Command('say Return to the Basin Warden. You are ready for what lies south of here.', 13.5);

    // Teach throw-stone spell and advance to graduate step
    user.LearnSpell("throw-stone");
    user.GiveQuest("1-graduate");

    return false;
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var elder = room.GetMob(elderMobId, true);
        if ( elder != null ) {
            if ( user.HasQuest("1-magic") && !user.HasQuest("1-graduate") ) {
                elder.Command('say The moons have been patient. I can afford the same courtesy.');
            } else if ( user.HasQuest("1-graduate") ) {
                elder.Command('say You have what you came for. The Warden waits at the south gate.');
            } else {
                elder.Command('say Complete the other trials first. The Fold rewards preparation.');
            }
        }
        return false;
    }
    return false;
}

function onExit(user, room) {
}

function onLoad(room) {
}
