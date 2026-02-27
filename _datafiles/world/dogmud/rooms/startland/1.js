
mapSignData = "";

// Generic Command Handler
function onCommand(cmd, rest, user, room) {

    if (cmd != "look" && cmd != "read" ) {
        return false;
    }
    
    if ( rest.substr(rest.length - 3) == "map" || rest.substr(rest.length - 4) == "sign" ) {
      
        SendUserMessage(user.UserId(), "You look at the map nailed to the sign.");
        SendRoomMessage(room.RoomId(), user.GetCharacterName(true)+" looks at the map nailed to the sign.", user.UserId());

        // Load the cached map, or re-generate and cache it if it's not there
        if ( mapSignData == "" ) {
            mapSignData = GetMap(room.RoomId(), 1, 22, 38, "Map of Startland", false, String(room.RoomId())+",×,Here");
        }

        // Send the map to the user.
        SendUserMessage(user.UserId(), mapSignData);

        return true;
    }
    
    return false;
}

// Executes when the room first loads.
function onLoad(room) {
    // Just running this to pre-cache the map so that if someone looks at the map it won't time out
    mapSignData = GetMap(room.RoomId(), 1, 22, 38, "Map of Startland", false, String(room.RoomId())+",×,Here");
}
