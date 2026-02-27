# World Building

This section contains instructions on how to add content to your world, mostly using in-game admin commands.
Not everything is possible via admin commands, and more advanced building may require editing the `.yaml` datafiles for a given room, item, mob, etc.

### Creating Your Own Zone
Zones are essentially a collection of rooms and will help to organize your room creation process.

#### Step 1: Define the Zone

Use the `build zone` command to create a new zone:
   ```
   build zone "My Custom Zone"
   ```
   This will automatically create an empty room within the zone.

#### Step 2: Configure Zone Properties

1. Retrieve the zone configuration information using:
   ```
   zone info
   ```
---

### Creating Rooms, Exits, and Defining Nouns

#### Step 1: Create a Room

1. Move to the desired zone using the `room [room #]` command (e.g., `room 1`).
2. Set properties for the room:
   - To set a title: `room set title "A Castle Drawbridge"`
   - To set a description: `room set description "You see a drawbridge."`
   - To set idle messages: `room set idlemessages "The wind blows.;The sand falls."`
3. Verify or retrieve room information using:

```
room info
```

Make sure to note down the room number, as it will help you navigate back to it quickly later.

#### Step 2: Add Exits

1. Create an exit linking rooms:
   ```
   room exit [exit_name] [room_id]
   ```
   Example: `room exit west 159`
2. Rename an existing exit (non-numeric names only):
   ```
   room exit edit [exit_name] [new_exit_name]
   ```
   Example: `room exit edit climb jump`
3. Toggle the secrecy of an exit:
   ```
   room secretexit [exit_name]
   ```
   Example: `room secretexit south`

#### Step 3: Define Nouns
To add more detail to your environment, you may choose to give certain nouns their own description.

1. Add or overwrite nouns in the room:
   ```
   room noun [name] [description]
   ```
   Example: `room noun chair "A wooden chair with intricate carvings."`
2. List all nouns in the room using:
   ```
   room nouns
   ```

---

### Creating a MOB/NPC 

1. Use the `mob create` command to start the interactive tutorial for creating a new MOB. Follow the prompts to define the name, description, and other properties.
2. Once created, spawn the NPC into a room with
   ```
   mob spawn [name]
   ```
   Replace `[name]` with the name of your NPC to place it in the current room.&#x20;
3. Once you've tested your mob, be sure to add the mob to the room's YAML configuration file; otherwise, the NPC will be lost during server cleanup processes. For example, in your room YAML file, you can add the following:

```yaml
spawninfo:
- mobid: 2
  message: A town guard emerges from a nearby building.
  idlecommands:
  - say did you know there's a sign in the Townsquare with a map of the area?

```
---
### Maintaining and Restarting the Server

1\. To stop the running server for maintenance or to restart it, press `Ctrl + C` in the terminal where the server is running. This will safely terminate the process.

2\. Type ''go run .'' again to restart the server.

---
### Starting your own empty MUD world
Now that you've got the basics down, it's time to start a fresh world and begin your creation journey. 
The ``_datafiles/world`` folder has a folder called ``empty`` inside of it. Make a copy of that folder and give it your own name (i.e. ``sudo cp -r empty/ myworld/``).
Then edit your ``_datafiles/config.yaml`` ``DataFiles:`` field to point to your new world folder.

## Updating from master branch
To update your local GoMUD installation when new updates are available on the master branch of the GitHub repository:

	cd GoMud

	git pull origin master

	go build

This will fetch the latest updates and rebuild the application.

