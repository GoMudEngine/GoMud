// Market Street - Room 108
// Merchant Adela teaches buy/sell/inventory/equip commands.
// Sub-steps: shopping_arrive -> shopping_buy -> shopping_equip

const merchantMobId = 63;

var buyAttempted = false;
var buyTicks = 0;
var equipAttempted = false;
var equipTicks = 0;

function onEnter(user, room) {

    // Only trigger tutorial if player has completed mutation step but not shopping_arrive
    if ( !user.HasQuest("1-mutation") || user.HasQuest("1-shopping_arrive") ) {
        return true;
    }

    var merchant = room.GetMob(merchantMobId, true);
    if ( merchant == null ) {
        return true;
    }

    merchant.Command('emote looks up from the counter as you approach.', 1.0);
    merchant.Command('say New arrival. Good. Let me show you how this works.', 2.0);
    merchant.Command('say Type <ansi fg="command">list</ansi> to see what I carry. Every merchant in Gaius uses the same command.', 3.5);
    merchant.Command('say When you find something you want, type <ansi fg="command">buy [item name]</ansi>. You can buy a sharp stick for three gold, a cotton shirt for three, or a small red potion for five.', 5.5);
    merchant.Command('say The Academy gave you a small stipend when you arrived. It should cover at least one purchase.', 7.0);
    merchant.Command('say Go ahead. Type <ansi fg="command">list</ansi>, then <ansi fg="command">buy</ansi> something.', 8.5);

    user.GiveQuest("1-shopping_arrive");

    return true;
}

function onCommand(cmd, rest, user, room) {

    // Detect buy command
    if ( cmd == "buy" ) {
        if ( user.HasQuest("1-shopping_arrive") && !user.HasQuest("1-shopping_buy") ) {
            buyAttempted = true;
            buyTicks = 0;
        }
    }

    // Detect wield or wear command (aliases resolve to "equip")
    if ( cmd == "equip" ) {
        if ( user.HasQuest("1-shopping_buy") && !user.HasQuest("1-shopping_equip") ) {
            equipAttempted = true;
            equipTicks = 0;
        }
    }

    if ( cmd == "talk" ) {
        var merchant = room.GetMob(merchantMobId, true);
        if ( merchant != null ) {
            if ( user.HasQuest("1-shopping_buy") && !user.HasQuest("1-shopping_equip") ) {
                merchant.Command('say You bought it. Now put it on. Type <ansi fg="command">wield [weapon]</ansi> or <ansi fg="command">wear [armor]</ansi>.');
            } else if ( user.HasQuest("1-shopping_arrive") && !user.HasQuest("1-shopping_buy") ) {
                merchant.Command('say Type <ansi fg="command">list</ansi> to see what I have, then <ansi fg="command">buy [item]</ansi>.');
            } else if ( user.HasQuest("1-shopping_equip") ) {
                merchant.Command('say Need something? Type <ansi fg="command">list</ansi> to see what I have.');
            } else {
                merchant.Command('say Come back after you have spoken with the Priest in the Academy Hall.');
            }
        }
        return false;
    }
    return false;
}

function onIdle(room) {

    // Handle buy detection
    if ( buyAttempted ) {
        buyTicks++;
        if ( buyTicks >= 2 ) {
            buyAttempted = false;
            buyTicks = 0;

            var players = room.GetPlayers();
            if ( players != null ) {
                for ( var i = 0; i < players.length; i++ ) {
                    var player = players[i];
                    if ( player.HasQuest("1-shopping_arrive") && !player.HasQuest("1-shopping_buy") ) {
                        var merchant = room.GetMob(merchantMobId, true);
                        if ( merchant != null ) {
                            merchant.Command('say Good choice. Now put it to use.', 0.5);
                            merchant.Command('say Type <ansi fg="command">inventory</ansi> -- or just <ansi fg="command">i</ansi> -- to see what you are carrying.', 2.0);
                            merchant.Command('say Weapons: type <ansi fg="command">wield [weapon]</ansi>. Armor and clothing: type <ansi fg="command">wear [item]</ansi>. To take something off, type <ansi fg="command">remove [item]</ansi>.', 3.5);
                            merchant.Command('say Go ahead. Equip what you bought.', 5.0);
                        }
                        player.GiveQuest("1-shopping_buy");
                    }
                }
            }
        }
    }

    // Handle equip detection
    if ( equipAttempted ) {
        equipTicks++;
        if ( equipTicks >= 2 ) {
            equipAttempted = false;
            equipTicks = 0;

            var players = room.GetPlayers();
            if ( players != null ) {
                for ( var i = 0; i < players.length; i++ ) {
                    var player = players[i];
                    if ( player.HasQuest("1-shopping_buy") && !player.HasQuest("1-shopping_equip") ) {
                        var merchant = room.GetMob(merchantMobId, true);
                        if ( merchant != null ) {
                            merchant.Command('say That is commerce. Simple enough.', 0.5);
                            merchant.Command('say I also buy things. If you have something you do not need, type <ansi fg="command">sell [item]</ansi> while you are here.', 2.0);
                            merchant.Command('say One more thing: type <ansi fg="command">status</ansi> to see your full character sheet -- stats, health, stamina, everything.', 3.5);
                            merchant.Command('say The Combat Trainer is in the Training Yard. From Town Square, go north then east.', 5.0);
                        }
                        player.GiveQuest("1-shopping_equip");
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
