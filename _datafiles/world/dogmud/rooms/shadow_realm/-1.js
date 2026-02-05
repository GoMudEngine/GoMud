
// If there is no book here, add the book item
function onEnter(user, room) {
    
    user.SendText('  <ansi fg="red">To get started, type <ansi fg="command">look</ansi> or <ansi fg="command">start</ansi>.</ansi>');
    user.SendText('');

    return true;
}
