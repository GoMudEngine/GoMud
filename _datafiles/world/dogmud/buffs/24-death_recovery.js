

// Invoked every time the buff is triggered (see roundinterval)
function onTrigger(actor, triggersLeft) {

    healAmt = actor.AddHealth(UtilDiceRoll(1, 10));
    manaAmt = actor.AddMana(UtilDiceRoll(1, 10));

    // Restore stamina and conviction too — without this, players bleed out
    // on those pools in the shadow realm and get stuck in a death loop.
    staminaAmt = actor.AddStamina(UtilDiceRoll(3, 10));
    convictionAmt = actor.AddConviction(UtilDiceRoll(3, 10));

    if ( healAmt > 0 && manaAmt > 0 ) {
        SendUserMessage(actor.UserId(),     'The shadow realm heals you for <ansi fg="healing">'+String(healAmt)+' damage</ansi> and restores <ansi fg="mana-100">'+String(manaAmt)+' mana</ansi>!');
    } else if ( healAmt > 0 ) {
        SendUserMessage(actor.UserId(),     'The shadow realm heals you for <ansi fg="healing">'+String(healAmt)+' damage</ansi>!');
    } else if ( manaAmt > 0 ) {
        SendUserMessage(actor.UserId(),     'The shadow realm restores <ansi fg="mana-100">'+String(manaAmt)+' mana</ansi>!');
    }

    if ( healAmt > 0 || manaAmt > 0 ) {
        SendRoomMessage(actor.GetRoomId(),  actor.GetCharacterName(true)+' is recovering from a recent death.', actor.UserId());
    }
}
