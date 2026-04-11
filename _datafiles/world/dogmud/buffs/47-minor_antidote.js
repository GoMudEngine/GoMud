
// Invoked when the buff is first applied to the player.
function onStart(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var healAmt = Math.floor(maxHP * 0.05);
    if (healAmt < 1) { healAmt = 1; }
    actor.AddHealth(healAmt);

    actor.RemoveBuff(39);
    actor.RemoveBuff(40);
}
