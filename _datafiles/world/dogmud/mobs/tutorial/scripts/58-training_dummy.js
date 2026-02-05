const teacherMobId = 57;

function onDie(mob, room, eventDetails) {

    room.SendText( mob.GetCharacterName(true) + " crumbles to dust." );

    var teacherMob = room.GetMob(teacherMobId, true);
    if ( teacherMob != null ) {
        teacherMob.Command('say You did it! As you can see you gain <ansi fg="experience">experience points</ansi> for combat victories.');
        teacherMob.Command('say Now head <ansi fg="exit">west</ansi> to complete your training.', 2.0);
    }
}
