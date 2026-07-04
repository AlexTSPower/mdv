# mdv

Terminal markdown viewer with GitHub-style rendering. Built with [Charm](https://charm.sh/).

![demo](https://github.com/AlexTSPower/mdv/assets/demo.gif)

## Install

```bash
brew install AlexTSPower/tap/mdv
```

Or with Go:

```bash
go install github.com/AlexTSPower/mdv@latest
```

## Usage

```bash
mdv              # open file browser at current directory
mdv ./path/repo  # open file browser at specified path
```

## Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Scroll up / move cursor up |
| `↓` / `j` | Scroll down / move cursor down |
| `Enter` | Open selected file |
| `b` | Toggle sidebar |
| `Tab` | Cycle focus between browser and viewer |
| `i` | Open current file in `$EDITOR` |
| `q` / `Ctrl+C` | Quit |

## Build from source

```bash
git clone https://github.com/AlexTSPower/mdv
cd mdv
go build -o mdv .
```

## License

MIT
