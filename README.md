# attyctl

A lightweight controller for handling the configurations of several terminal
applications:

- macOS’s native Terminal
- Alacritty’s TOML settings (still a work in progress)
- WezTerm’s configuration (not yet)

This project forms a component of an evaluation of three alternatives,
concentrating mainly on their energy usage to pinpoint the most eco‑friendly
choice. 🍃  

Its inception was a spur‑of‑the‑moment idea—I was looking for a simple, fast
way to manage the terminal font.

## Dependencies

- Use [fzf](https://github.com/junegunn/fzf) to get an interactive,
  fuzzy‑search style font‑selection experience.  
- Use `system_profiler` to retrieve a list of all installed font family names
  on macOS (see `man system_profiler`).

## Setup

The setup step runs only when the `~/.local/fonts.txt` file is missing, and you
need to execute `attyctl -setup` each time you want to use newly installed
fonts.

## Alacritty support

Currently, testing has been limited to altering
the [font.normal].family property.

I've created a tool that lets me adjust the font on the fly. Since I'm on a
Mac, I use `systemprofiler` to generate a JSON list of all installed fonts,
parse it for the family names, and cache that list in `~/.local/fonts.txt`.

The setup step runs only when the `~/.local/fonts.txt` file is missing, and you
need to execute `attyctl -setup` each time you want to use newly installed
fonts.
