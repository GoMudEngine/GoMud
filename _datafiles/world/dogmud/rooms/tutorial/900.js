

const allowed_commands = ["help", "broadcast"];
const teach_commands = ["look", "look orb", "look", "look east", "east"];
const teacherMobId = 57;
const teacherName = "Orb of Vision";

var commandNow = 0; // Which command they are on
var canGoEast = false;




// Generic Command Handler
function onCommand(cmd, rest, user, room) {

    ignoreCommand = false;

    teacherMob = getTeacher(room);

    var extraDelay = 0;
    // Make sure they are only doing stuff that's allowed.

    if ( cmd == "east" && !canGoEast ) {
        teacherMob.Command("say Not so hasty! Lets finish the basics before you leave this area.");
        extraDelay = 1.0;
        ignoreCommand = true;
    }

    fullCommand = cmd;
    if ( rest.length > 0 ) {
        fullCommand = cmd + ' ' + rest;
    }

    if ( teach_commands[commandNow] == fullCommand ) {
        
        teacherMob.Command("say Good job!", 1.0);

        extraDelay = 1.0;

        if ( fullCommand == "look orb" ) {
            teacherMob.Command('say As you can see, looking at me shows you a description and some information about what I\'m carrying.', 2.0);
            extraDelay = 2.0;
        }

        if ( fullCommand == "look east" ) {
            teacherMob.Command('say Looking into exits like that shows you what (or who) is in a room before you visit it.', 2.0);
            teacherMob.Command('say Later when you find objects, you can look at them in the same manner.', 3.0);
            teacherMob.Command('say It\'s always worth trying to look at something you\'re curious about, just in case.', 4.0);
            teacherMob.Command('emote considers for a moment.', 5.0);
            extraDelay = 7.0;
        }

        commandNow++;
    } else {

        if ( allowed_commands.includes(cmd) || teach_commands.slice(0, commandNow).includes(cmd) ) {
            return false;
        }
        
        ignoreCommand = true;
    }

    switch (commandNow) {
        case 0: 
            teacherMob.Command('say The first thing you need to learn is how to inspect your surroundings', extraDelay+1.0);
            teacherMob.Command('say type <ansi fg="command">look</ansi> and hit enter to see a description of the area you are in.', extraDelay+2.0);
            break;
        case 1:
            teacherMob.Command('say You can also look at creatures or people in the room.', extraDelay+1.0);
            teacherMob.Command('say type <ansi fg="command">look orb</ansi> to look at me, ' + teacherMob.GetCharacterName(true) + '.', extraDelay+2.0);
            break;
        case 2:
            teacherMob.Command('say Try the <ansi fg="command">look</ansi> command again, but this time, pay attention to any <ansi fg="exit">exits</ansi>.', extraDelay+1.0);
            break;
        case 3:
            teacherMob.Command('say Did you notice there is an exit to the <ansi fg="exit">east</ansi>?', extraDelay+1.0);
            teacherMob.Command('say type <ansi fg="command">look east</ansi> to look into the <ansi fg="exit">east</ansi> room.', extraDelay+2.0);
            break;
        case 4:
            canGoEast = true;
            teacherMob.Command('say It\'s time to move on to the next thing you\'ll learn about.', extraDelay+1.0);
            teacherMob.Command('say type <ansi fg="command">east</ansi> to travel through the <ansi fg="command">east</ansi> exit.', extraDelay+2.0);
            break;
        default:
            teacherMob.Command('say It\'s time to move on to the next thing you\'ll learn about.', extraDelay+1.0);
            teacherMob.Command('say type <ansi fg="command">east</ansi> to travel through the <ansi fg="command">east</ansi> exit.', extraDelay+2.0);
            break;
    }
    
    return ignoreCommand;
}




// If there is no book here, add the book item
function onEnter(user, room) {
    teacherMob = getTeacher(room);
    canGoEast = false;
    commandNow = 0;

    sendWorkingCommands(user);

    teacherMob.Command('emote appears in a ' + UtilApplyColorPattern("flash of light!", "glowing"));
    
    teacherMob.Command('say Welcome to the Newbie School!', 1.0);
    teacherMob.Command('say I\'ll give you some tips to help you get started.', 2.0);
    teacherMob.Command('say In this area you\'ll learn the basics of inspecting your environment with the <ansi fg="command">look</ansi> command.', 3.0);
    teacherMob.Command('say type <ansi fg="command">look</ansi> and hit enter to see a description of the area you are in.', 5.0);

    return true;
}

function onExit(user , room) {
    // Destroy the guide (cleanup)
    destroyTeacher(room);
}

function onLoad(room) {
    canGoEast = false;
    commandNow = 0;
}

function getTeacher(room) {
    var mobActor = room.GetMob(teacherMobId, true);
    mobActor.SetCharacterName(teacherName);
    return mobActor;
}

function destroyTeacher(room) {
    var mobActor = room.GetMob(teacherMobId);
    if ( mobActor != null ) {
        mobActor.Command(`suicide vanish`);
    } 
}

function sendWorkingCommands(user) {

    ac = [];
    unlockedCommands = teach_commands.slice(0, commandNow);

    for (var i in allowed_commands ) {
        ac.push(allowed_commands[i]);
    }
    
    for ( i in unlockedCommands ) {
        ac.push(unlockedCommands[i]);
    }
    
    user.SendText("");
    user.SendText("");
    user.SendText('    <ansi fg="red">NOTE:</ansi> Most commands have been <ansi fg="203">DISABLED</ansi> and <ansi fg="203">WILL NOT WORK</ansi> until you <ansi fg="51">COMPLETE THIS TUTORIAL</ansi>!');
    //user.SendText('          The commands currently available are: <ansi fg="command">'+ac.join('</ansi>, <ansi fg="command">')+'</ansi>');
    user.SendText("");
    user.SendText("");

}
