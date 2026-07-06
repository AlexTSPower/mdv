# stackreader

Terminal markdown viewer with GitHub-style rendering. Built with [Charm](https://charm.sh/).

![demo](https://github.com/AlexTSPower/StackReader/assets/demo.gif)

## Install

```bash
brew install AlexTSPower/tap/stackreader
```

Or with Go:

```bash
go install github.com/AlexTSPower/StackReader@latest
```

## Usage

```bash
stackreader              # open file browser at current directory
stackreader ./path/repo  # open file browser at specified path
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
git clone https://github.com/AlexTSPower/StackReader
cd StackReader
go build -o stackreader .
```

## License

MIT
