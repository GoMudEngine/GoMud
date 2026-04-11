function onTrigger(actor, triggersLeft) {
    var maxHP = actor.GetHealthMax();
    var dmg = Math.floor(maxHP * (0.06 + Math.random() * 0.04));
    if (dmg < 2) dmg = 2;
    actor.AddHealth(-dmg);
    SendUserMessage(actor.UserId(), 'The toxic fumes burn your lungs!');
    SendRoomMessage(actor.GetRoomId(), actor.GetCharacterName(true) + ' chokes and gasps in the toxic cloud.', actor.UserId());
}
