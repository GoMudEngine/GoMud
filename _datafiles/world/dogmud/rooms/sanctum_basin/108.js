// Market Street - Room 108
// Merchant Adela teaches buy/sell/inventory/equip commands.

const merchantMobId = 63;

function onEnter(user, room) {

    // Only trigger tutorial if player has completed mutation step but not shopping
    if ( !user.HasQuest("1-mutation") || user.HasQuest("1-shopping") ) {
        return true;
    }

    var merchant = room.GetMob(merchantMobId, true);
    if ( merchant == null ) {
        return true;
    }

    merchant.Command('emote looks up from the counter as you approach.', 1.0);
    merchant.Command('say New arrival. Good. Let me show you how this works.', 2.0);
    merchant.Command('say Type <ansi fg="command">list</ansi> to see what I carry. Every merchant in Gaius uses the same command -- if they have a shop, that is how you see it.', 3.5);
    merchant.Command('say When you find something you want, type <ansi fg="command">buy [item name]</ansi>. You can buy a sharp stick for three gold, a cotton shirt for three, or a small red potion for five.', 5.5);
    merchant.Command('say The Academy gave you a small stipend when you arrived. It should cover at least one purchase.', 7.0);
    merchant.Command('say After you buy something, type <ansi fg="command">inventory</ansi> -- or just <ansi fg="command">i</ansi> -- to see what you are carrying.', 8.5);
    merchant.Command('say To check what you have equipped, type <ansi fg="command">eq</ansi>.', 10.0);
    merchant.Command('say Weapons: type <ansi fg="command">wield [weapon]</ansi> to hold it. Armor and clothing: type <ansi fg="command">wear [item]</ansi>. To take something off, type <ansi fg="command">remove [item]</ansi>.', 11.5);
    merchant.Command('say I also buy things. If you have something you do not need, type <ansi fg="command">sell [item]</ansi> while you are in this room.', 13.5);
    merchant.Command('say One more thing: type <ansi fg="command">status</ansi> to see your full character sheet -- stats, health, stamina, everything. Worth a look before you head into a fight.', 15.0);
    merchant.Command('say That is commerce. Simple enough.', 17.0);
    merchant.Command('say The Combat Trainer is in the Training Yard. From Town Square, go north then east. Or use the <ansi fg="command">map</ansi> command if you get turned around.', 18.5);

    user.GiveQuest("1-shopping");

    return true;
}

function onCommand(cmd, rest, user, room) {
    if ( cmd == "talk" ) {
        var merchant = room.GetMob(merchantMobId, true);
        if ( merchant != null ) {
            if ( user.HasQuest("1-shopping") ) {
                merchant.Command('say Need something? Type <ansi fg="command">list</ansi> to see what I have. Confluence prices went up again -- happens every time the river runs low and the supply caravans get delayed.');
            } else if ( user.HasQuest("1-mutation") ) {
                merchant.Command('say First time here? Type <ansi fg="command">list</ansi> to see my wares. Buy something and equip it.');
            } else {
                merchant.Command('say Come back after you have spoken with the Priest in the Academy Hall.');
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
