# tbit
tbit is a chat server exercise.
It supports rooms, logging to a local file, reading from a local configuration file.

Commands:
* /help - this text
* /exit - close your connection
* /quit - close your connection
* /user <username> - change your username
* /rooms - lists rooms that have been created
* /join <room> - joins a new room
* /leave <room> - leaves a room you are in
* /list - lists which rooms you are currently in
* /say <room> <message> - used to send a message to a specific room

An example config file is in the repo as `tbit.conf.example`.

----

This implementation creates buffered channels per connection for output handling.
Rooms send messages to each connection's output channel.
The size of the buffer should be tuned using real data.
Currently it times out after a second if the buffer is full.
But for some use cases it might be best to drop the output on the floor.

Input that isn't a command is announced to all rooms the connection is in.

Connection IDs are currently integers. This should be changed if we expect more total connections per instance than integers can hold. Either reusing IDs, using a larger type, or removing the need for IDs entirely would fix this limitation.

The only 3rd-party package used is github.com/pelletier/go-toml for the config file parsing.
I find TOML to be better than JSON for config files that are written manually.

Todo
----
* Command to see who is in a room.
* Ring buffer of each room's output to display when joining a room.
* Per room logging.
* Better user messaging for edge cases, such as announcing when not in any rooms.
* Add some hardcoded values to the config file.
  * buffer sizes
  * timeouts
  * welcome and help messages
  * default room name

