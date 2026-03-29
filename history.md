# Conversation History

- User started by asking to read `plan.md`.
- `plan.md` requested creating a TUI application to output the best updated torrent trackers for different kinds (all, http, https, ip, best). The plan explicitly requested querying the user for their preferred programming language instead of just using Python.
- I queried the user for their language preference.
- User selected **Go** and requested that all future conversation be logged to `history.md`.
- I saved the conversation to `history.md` and drafted an Implementation Plan for a Go TUI application.
- User reviewed and approved the plan with two specific requests: name the Go module `tracker`, and include functionality to display the trackers, copy them to the clipboard, and save them to a `.txt` file.
- I drafted the Go code but Go wasn't explicitly installed on the system (CachyOS), so I asked the user to install it via Pacman.
- User installed `go` and modified `plan.md` to specify using the "Tokyo Night high contrast color scheme".
- I applied the Tokyo Night high contrast colors to the `lipgloss` styling components in the application.
- Initialized Go module and compiled the application successfully! Complete walkthrough was generated.
- User requested adding explicit support for Wayland clipboards. I added a fallback copying mechanism explicitly tied to Wayland's `wl-copy` utility.
- User requested centering the entire TUI layout, adding an ASCII banner at the top, and writing a "Made with love by Github : roxxadiiii" footer at the bottom. I updated the View and lipgloss formatting to center the UI globally inside the user's terminal.
- User successfully copied trackers using Wayland clipboards!
- User updated `plan.md` requesting the app to be structured like a module in a separate folder and compiling the final executable to the project root directory. I moved `main.go` into `cmd/tracker/main.go` and compiled the binary to the project root.
- User then requested renaming the executable binary to `tor-tracker` and isolating all the project's source code and configuration files into their own `tracker` subdirectory. I executed a script to move all items except `.git` and the executable `tor-tracker` into the new `tracker` wrapper folder!
- User initiated Phase 2 by modifying `plan.md` to request a new torrent search engine TUI, contained entirely within a new `search` folder, following the Tokyo Night color scheme, and showing detailed magnet metadata.
- I drafted a new Implementation Plan for Phase 2 encompassing text input flows, table layouts, and public torrent API endpoints (APIBay and YTS).
