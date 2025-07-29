# Configuration

Configuration lies in ~/.config/conan/settings.ini

## Client parameters

Clients can be various, putty, openssh, some terminals with ssh builtin and so on...

Here are some ini keys for configuration

* ssh_client = external (default), putty (windows) builtin (not yet)
* linux_ssh
* linux_rdp
* linux_winbox
* windows_ssh
* windows_rdp
* windows_winbox
* darwin_ssh
* darwin_rdp
* darwin_winbox

Each key accepts template based parameters (Excluding ssh_client), which also can be scripted using {{- if}} statements and many more macro functions,
here are the basic parameters that can be used:

```
{{.ID}}            -> Unique identifier of the server in the memory (each program startup it's different)
{{.SourcePath}}    -> Server's yml file, where server information is saved
{{.Host}}          -> Server hostname
{{.IP}}            -> Server IP address
{{.User}}          -> Username for server
{{.Password}}      -> Password for connection if specified
{{.PrivateKey}}    -> Server private key if specified
{{.Port}}          -> Server port
{{.Description}}   -> Server description
{{.Type}}          -> Server type (eg. ssh, rdp, winbox)
{{.Tags}}          -> Server tags (separated by commas)
{{.Home}}          -> Users home directory (~/ on Unix, %userprofile% on Windows)
{{.AppDir}}        -> Application directory where the binarie lies
{{.ConfigDir}}     -> Application configuration directory (Default ~/.config/conan on Unix)
{{.DefaultKey}}    -> Default ssh key specified in settings.ini [General] defaultsshkey =
```

## Servers definitions

There can be several yml files located in ~/.config/conan or in it's program directory, at the program startup it automatically search and load yml files.
You can define separate sync settings for them. For example one for home and one for work. It will sync in separate gists, you can also share the gist with your collegues then. It will be useful for SySadmins in large teams, where it needs to share many connections to servers.


## Hyprland bindings

Because the built-in global hotkeys does not work in Wayland at the time, we currently will use the external hotkey mechanism on Hyprland
it will execute the:
```
./conan --tray --show
```

Edit file **~/.config/hypr/UserConfigs/UserKeybinds.conf** and add this line:
```
bind = CTRL, SPACE, exec, $HOME/Projects/sshexperiment/conan -tray -show
```

## Sync

Sync servers list between computers using github gist system, 
this way it stores servers list in private gist. 
1. On github navigate: Settings → Developer  Settings → Personal Access Token  → Tokens (classic)
2. Generate a classic token with gist scope
3. Copy it to **gistsecret**
4. Nagitate https://gist.github.com/ and create secret gist
5. Copy its id after **gist:** and put to **gistid**.

Enable it in configuration:
```
[General]
sync = true

[gist one.yml]
gistid  = gist_id
gistsec = classic access token
enckey  = encyption key
```

# Configuration example

Configuration example consists of example parameters for three main operating systems, Linux, Windows, MacOS

```
[General]
enckey         = ...
ssh_client     = external # builtin, external, putty
darwin_ssh     = {{.AppDir}}/scripts/osahelper --client ssh --user {{.User}} --host {{.IP}} {{- if .Password}} --password {{.Password}}{{end}} {{- if .Port}} --port {{.Port}}{{end}} {{- if .PrivateKey}} -i {{.Home}}/.ssh/{{.PrivateKey}}{{else}} -i {{.DefaultKey}}{{end}}
darwin_rdp     = /opt/homebrew/bin/xfreerdp /u:{{.User}} /v:{{.IP}} /p:{{.Password}} /cert:ignore /log-level:ERROR +dynamic-resolution /size:1200x800 /clipboard
darwin_winbox  = /Applications/WinBox.app/Contents/MacOS/WinBox {{.IP}} {{.User}} {{.Password}}
windows_ssh    = C:\programs\putty\putty.exe -ssh {{.User}}@{{.IP}} {{- if .Port}} -P {{.Port}}{{end}} {{- if .Password}} -pw {{.Password}}{{end}} {{- if .PrivateKey}} -i {{.Home}}/.ssh/{{.PrivateKey}}{{else}} -i {{.DefaultKey}}{{end}}
windows_rdp    = mstsc /v:{{.IP}} /admin
windows_winbox = C:\programs\winbox\winbox64.exe {{.IP}} {{.User}} {{.Password}}
linux_ssh      = kitty ssh {{.User}}@{{.IP}} {{- if .Port}} -p {{.Port}}{{end}} {{- if .PrivateKey}} -i {{.Home}}/.ssh/{{.PrivateKey}}{{else}} -i {{.DefaultKey}}{{end}}
linux_rdp      = xfreerdp3 /u:%u /v:%ip /p:%p% /cert:ignore /f /log-level:ERROR
linux_winbox   = wine %H/.bin/winbox64.exe {{.IP}} {{.User}} {{.Password}}
sync           = true
ignore         =
defaultsshkey  = {{.AppDir}}/.ssh/identity

[ServersTable]
disabletooltips    = false
disablerowtooltips = false
HostColumn         = 180
TypeColumn         = 80
IpColumn           = 115
UserColumn         = 80
DescriptionColumn  = 120
TagsColumn         = 150
SourceColumn       = 100
AvailabilityColumn = 85

[gist one.yml]
gistid  = 
gistsec = 
enckey  =

[gist two.yml]
gistid  =
gistsec =
enckey  = 
```

## SSH With password on UNIX

```
brew install esolitos/ipa/sshpass
```

enckey can be specified globally or per gist sync, if gist sync uses different key, when all passwords and file's encyption will use that key else it will use the global key

# Command line options

examples.:

./conan --db mano.yml --chgkey "new_key" # Changes the database encryption key for passwords
./conan importsettings --file ~/conan-settings-20250710-022525.cnn # Imports settings file (protected by the password)


