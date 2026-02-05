
const allowed_commands = ["help", "broadcast", "look", "status", "inventory", "experience", "conditions"];
const teach_commands = ["equip stick", "attack dummy", "west"];
const teacherMobId = 57;
const dummyMobId = 58;
const teacherName = "Orb of Violence";
const firstItemId = 10001;

var commandNow = 0; // Which command they are on



// Generic Command Handler
function onCommand(cmd, rest, user, room) {

    ignoreCommand = false;

    teacherMob = getTeacher(room);

    var extraDelay = 0;
    // Make sure they are only doing stuff that's allowed.

    if ( cmd == "south" && !canGoSouth ) {
        teacherMob.Command("say Not so hasty! Lets finish up here before you leave the area.");
        ignoreCommand = true;
    }

    fullCommand = cmd;
    if ( rest.length > 0 ) {
        fullCommand = cmd + ' ' + rest;
    }

    if ( teach_commands[commandNow] == fullCommand ) {
        
        teacherMob.Command("say Good job!", 1.0);

        extraDelay = 1.0;

        if ( cmd == "equip stick" ) {
            teacherMob.Command('say Check it out! If you type <ansi fg="command">status</ansi> you\'ll see the stick is equipped!', 2.0);
            extraDelay = 2.0;
        }

        if ( cmd == "inventory" ) {
            teacherMob.Command('say Hmm, it doesn\'t look like you\'re carrying much other than that sharp stick.', 2.0);
            teacherMob.Command('say Remember, you can <ansi fg="command">look</ansi> at stuff you\'re carrying any time you want.', 3.0);
            extraDelay = 3.0;
        }

        commandNow++;

        if ( cmd == "attack dummy" ) {
            return false;
        }

    } else {

        if ( allowed_commands.includes(cmd) || teach_commands.slice(0, commandNow).includes(cmd) ) {
            return false;
        }
        
        ignoreCommand = true;
    }

    switch (commandNow) {
        case 0:

            if ( !user.HasItemId(firstItemId) ) {
                itm = CreateItem(firstItemId);
                user.GiveItem(itm);
            }
            
            teacherMob.Command('say Go ahead and equip that sharp stick you\'ve got. Type <ansi fg="command">equip stick</ansi>.', extraDelay+1.0);
            break;
        case 1:

            getDummy(room);

            teacherMob.Command('say You may have noticed the <ansi fg="mobname">training dummy</ansi> here.', extraDelay+1.0);
            teacherMob.Command('say Go ahead and engage in combat by typing <ansi fg="command">attack dummy</ansi>.', extraDelay+2.0);
            break;
        case 2:
            // teacherMob.Command('say Head <ansi fg="exit">west</ansi> to complete your training.');
            break;
        default:
            break;
    }
    
    return ignoreCommand;
}




// If there is no book here, add the book item
function onEnter(user, room) {
    room.SetLocked("north", true);
    
    teacherMob = getTeacher(room);
    getDummy(room);

    sendWorkingCommands(user);

    teacherMob.Command('emote appears in a ' + UtilApplyColorPattern("flash of light!", "glowing"));
    
    teacherMob.Command('say It looks like it\'s time for the most dangerous part of your lesson!', 1.0);

    if ( !user.HasItemId(firstItemId) ) {
        itm = CreateItem(firstItemId);
        user.GiveItem(itm);
    }

    teacherMob.Command('say Go ahead and equip that sharp stick you\'ve got. Type <ansi fg="command">equip stick</ansi>.', 2.0);

    return true;
}

function onExit(user , room) {
    // Destroy the guide (cleanup)
    destroyTeacher(room);
    destroyDummy(room);
    canGoSouth = false;
    commandNow = 0;
}

function onLoad(room) {
    canGoSouth = false;
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

function getDummy(room) {
    return room.GetMob(dummyMobId, true);
}

function destroyDummy(room) {
    var mobActor = room.GetMob(dummyMobId);
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
