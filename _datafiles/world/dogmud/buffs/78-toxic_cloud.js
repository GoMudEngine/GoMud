function onStart(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'A cloud of choking fumes engulfs you!');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' is engulfed in a cloud of toxic fumes.', actor.UserId());
}

function onTrigger(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var dmg = Math.floor(maxHP * (0.04 + Math.random() * 0.03));
    if (dmg < 2) dmg = 2;
    actor.AddHealth(-dmg);
    SendUserMessage(actor.UserId(), 'The toxic fumes burn your lungs!');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' chokes and gasps in the toxic cloud.', actor.UserId());
}

function onEnd(actor, triggersLeft) {
    SendUserMessage(actor.UserId(), 'The toxic cloud finally dissipates.');
}
