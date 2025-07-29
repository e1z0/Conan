-- PORT TO QT

[X] when compiled with -H windowsgui does not write anything to debug.log
[X] port server table window to QT
[X] port spotlight type of search to QT (ported and improved with nice UI/UX)
[X] port settings window to QT
[X] port welcome dialog to QT
[X] port notes to QT
[X] port about window to QT
* hotkey is already registered shows in windows
* connect does not work on windows using putty as the client (builtin)
* change global icon on windows to represent conan
[X] after deleting note, all notes become with folder icons, possible cause is nw.treeWidget.Refresh()
* delete note window size and position from config file when note itself is deleted
[X] rearagement of the items via drag-drop in servers table window
* ability to import excel (detect columns before importing and show binding to real data structure if available)
* grouping when clicking on the columns in server table window
* ability to set jump hosts for all servers in the yml, also ability for other servers to use these jumphosts (build list on server edit)

* delete folder with subnotes from gist when deleting folder...
* after rework in cmdline options --chgkey  does not work
x ability to resize column positions in servers table window list -> NOT POSSIBLE AT THE TIME
* on the second note window something strange happends, when clicking view mode, it sometimes bugs out and only in edit mode the text is visible
* export note as pdf document
* look at the https://github.com/andydotxyz/slydes special implementation of markdown reader, maybe we can rip the ideas off it
* auto update ability
* sticky note remember last position and size... to note itself, lol
* update sticky note from the main notes window on the fly...

[X] after note deletion/creation in view mode, after switch modes from editor to view, the text becomes invisible
[X] when everything is encrypted on startup, it does not request to delete remote gist note on note deletion...
[X] command line switch which will import/export settings as in the GUI
[X] Encrypt password command and return encrypted password (will be used in scripting)
[X] Add ability to categoryze menu items
[X] Add tray menu items alphabetically
[X] Yml support implementation
[X] when updating servers from the servers window table update it globally (tray menu items also)
[X] tighten integration with some terminal, maybe iterm2, ability to open new tab, or window navigation that ties all active sessions?
[X] ping hotkey to ping all hosts on the list
[X] fix on win32 Opening external: C:\Users\justi/.config/conan does not open program path
[X] implement auto launch gui mode when .exe detected
[X] allow only single window of servers table
[X] servers table search box, focus on text input, submit on enter, search dialog is invoked by hotkey CTRL+F or CMD+F (on mac)
[X] export encrypted config directory using scrypt for KDF with high work factor (N=32768)? r=8, p=1 AES-256-GCM for authenticated encryption output format 16 bytes salt | 12 bytes nonce | ciphertext...
[X] import encrypted config file
[X] when adding new server and no server list is selected, it continues to add server no matter what, it needs to be fixed!
[X] unable to select first row on servers table window list
[X] gist remove button from the settings actually does nothing...
[X] when connecting from servers table window, it hangs and allows one connection only, need to make it with threading
[X] ignore does not work from settings.ini
[X] if one yml is ignored and sync is pushed to github gist error accours that shows no file found
[X] settings windows different settings for different OS (autodetect current OS)
[X] move build process from the root of project tree to ./src on MacOS to use the same FyneApp.toml
[X] ability to disable tooltips for server table in settings, also ability to enable tooltips on selected row
[X] when connecting from context menu using context menu "Connect" it does not fork new connection
[X] command line parameters templating system
[X] defaultsshkey should parse {{.AppDir}} and other variables...
[X] from tray icon options menu need to implement restart functionality
[X] export settings with notes included, all settings folder are now exported
[X] encrypt notes checkbox in settings near the gist configuration
[X] expert settings tab in settings window
[X] fix ping servers from servers table window, ping lib have been migrated to probing from prometheus-grafana project... needs extensive testing on windows
[X] wrap large lines in notes richtext editor
[X] delete gist when it's deleted from the notes
[X] when encrypted settings cannot push gist of servers
[X] when encrypted "show" from tray menu does not work, hotkey works
[X] settings are being saved as darwin os always, need os independent code
[X] some critical windows should run on the center of the screen

[X] - Fully implemented
?*  - Paritally implemented (UNTESTED)
*   - Not implemented yet
x   - Canceled
