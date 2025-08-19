# Conan – Cross-Platform Connection Manager

**Conan** is a fast, minimal, and extensible **connection manager** built with **Go** and **Qt 5**. Also there is CLI interface with a rich curses-based UI.

It offers a native system tray menu with fuzzy search and instant-launch capabilities for various remote access tools like **SSH**, **Winbox**, and **RDP**.

Conan is designed for system administrators, network engineers, and developers who manage hundreds of remote systems and need fast, keyboard-driven access with minimal UI.

---

## ✨ Features

- 🖥️ **Native system tray integration** (macOS, Windows, Linux)
- 🔍 **Spotlight-style launcher** with fuzzy search
- 🧠 **Protocol-aware connection manager**: supports `ssh`, `rdp`, `winbox`, and custom command formats
- 📝 **Sticky notes** system (Markdown + metadata),
- ☁️ **GitHub Gist sync** for note and server lists (encrypted or plaintext)
- 🔐 **YAML-based server configuration**, supports tags and metadata
- ⚙️ **Terminal embedding** (via external window)
- 🪄 **Global hotkey support** to open launcher instantly
- 🎛️ **Server manager GUI** to edit connections visually
- 🧑‍💻 **Terminal UI** (curses-based) for CLI access
- 📦 Cross-platform builds: `.app`, `.exe`, AppImage

---

## In action



https://github.com/user-attachments/assets/a831d66d-5eb8-4249-b27e-bc823fb8bb13

https://github.com/user-attachments/assets/420259d0-bd82-442d-a06b-c6acd48e0e06

https://github.com/user-attachments/assets/9ce2b8c3-ead9-48d2-b07f-eda6cc150dec

https://github.com/user-attachments/assets/21d72cb7-cc47-40a3-8a53-13bf5d9064e0



## 🚀 Usage Overview

Conan runs in the background with a tray icon. You can:
- Click the tray icon to see categorized connection entries
- Use a **global hotkey** to open the launcher
- Type a hostname, description or tag to **fuzzy search** your connections
- Press `Enter` to instantly launch your session with the correct protocol

---


## 🖥️ Terminal UI Mode (CLI)

Conan includes a **curses-style CLI interface** for terminal environments:

```
conan --tui
```

## 🔗 Supported Protocols

Conan understands the following protocols natively:

- `ssh://user@host`
- `rdp://hostname`
- `winbox://192.168.88.1`
- Custom launchers via YAML config (e.g., `tmux`, `telnet`, `mosh`, etc.)

The correct external tool is launched based on the protocol:
- `ssh` → `kitty`, `xterm`, or your defined terminal
- `rdp` → `xfreerdp`, `mstsc`
- `winbox` → `wine winbox.exe` or native

---

## 🧾 Notes System

Conan allows you to create **sticky notes** for each topic. Notes are:

- Written in **Markdown**
- Stored as `.md` files in a designated folder
- Include **YAML front-matter metadata**:


## 🔐 Security

Conan is designed for security-conscious system administrators and infrastructure engineers. It includes multiple layers of protection to keep credentials, notes, and configuration safe:

- 🔐 **Encrypted Configuration Files**  
  The main server list (`servers.yml`) can be **fully AES-256 encrypted**. This includes:
  - Server addresses
  - Protocols
  - Usernames
  - Tags and metadata

- 🔑 **Credentials Encryption**  
  Individual connection credentials (e.g., usernames, ports, passwords, secret keys) are securely encrypted and stored only locally. No plaintext credentials are left on disk.

- 📝 **Encrypted Notes**  
  Markdown-based sticky notes can be individually encrypted using the same symmetric encryption. The app decrypts them in-memory and never saves plaintext unless explicitly exported.

- 🔐 **Master Password Prompt**  
  On launch, Conan can optionally prompt the user for a **master password**. This password is required to:
  - Unlock the encrypted server list
  - Access private notes
  - Decrypt credentials

- ☁️ **Secure GitHub Gist Sync (Optional)**  
  If Gist sync is enabled:
  - Notes are encrypted before upload
  - Only a securely stored GitHub token is used
  - Private gists are used for syncing between machines
  - Version history is preserved locally (encrypted)

- 🧠 **Zero Background Communication**  
  - No telemetry, no hidden requests, no cloud dependencies
  - Conan is fully offline unless syncing to GitHub is explicitly triggered

> All encryption is performed using strong, modern AES-256-GCM in authenticated mode. Secrets are stored only in memory during the runtime session and are wiped from disk.


# Building

Read [Building.md](/doc/BUILDING.md)

# Configuration

Read [Configuration.md](/doc/CONFIGURATION.md)

# 3rd party software/material used

* [Golang](https://go.dev)

* [Qt 5 bindings for Go](https://github.com/mappu/miqt)
* [Qt 5.15 – Native GUI toolkit](https://www.qt.io)
* [Icons](https://icon.kitchen)
