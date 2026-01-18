package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/4sp1/attyctl/internal/fonts"
	"github.com/4sp1/must"
	"github.com/BurntSushi/toml"
)

type config map[string]map[string]any

func main() {
	setupAlacritty := flag.Bool("alacritty", false, "set up alacritty")
	setupTerminal := flag.Bool("terminal", true, "set up the native macOS Terminal")
	setupFlag := flag.Bool("setup", false, "rebuild cache for newly installed fonts")
	setupJson := flag.String("json", "", "path to export system_profile SPFontsDataType as JSON")
	setupFont := flag.Bool("font", true, "set up font family")
	setupFontFamily := flag.String("family", "", "font family name")
	setupFontStyle := flag.String("style", "", "font family name")
	flag.Parse()

	if !*setupFont {
		fmt.Println("Nothing to do. Bye.")
	}

	var fontList io.Reader
	{
		cache := fonts.NewCache(fonts.WithWriteSystemProfilerFontsTo(*setupJson))
		if *setupFlag {
			fontList = cache.Refresh()
		} else {
			fontList = cache.NewReader()
		}
	}

	font := fonts.Font{
		Family: *setupFontFamily,
		Style:  *setupFontStyle,
	}

	if len(font.Family) == 0 {
		cmd := exec.Command("fzf")
		cmd.Stdin = fontList
		var b bytes.Buffer
		cmd.Stdout = &b
		cmd.Stderr = os.Stderr
		mustHandle(cmd.Run)
		mustHandleError(json.NewDecoder(&b).Decode(&font))
	}

	if len(font.Style) == 0 {
		font.Style = "Regular"
	}

	switch {
	case *setupAlacritty:
		alacritty(font)
	case *setupTerminal:
		mustHandleError(terminal(*setupFontFamily))
	default:
		fmt.Println("Nothing to do. Bye.")
		os.Exit(2)
	}
}

var (
	exitHandler     = must.ExitHandler(1)
	mustHandle      = must.Handle(exitHandler)
	mustHandleError = must.HandleError(exitHandler)
	mustHaveString  = must.Have(must.ExitController[string](1))
	mustHaveFile    = must.Have(must.ExitController[*os.File](1))
)

func terminal(font string) error {
	cmd := exec.Command("osascript", "-e",
		`tell application "Terminal" to set the font name of window 1 to "`+font+`"`)
	return cmd.Run()
}

func alacritty(font fonts.Font) {
	home := mustHaveString(os.UserHomeDir())
	file := path.Join(home, ".config", "alacritty", "alacritty.toml")

	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			mustHandleError(os.MkdirAll(path.Dir(file), 0700))
			f := mustHaveFile(os.Create(file))
			defer mustHandle(f.Close)
			config := config{
				"font": map[string]any{
					"size": 14.0,
					"normal": map[string]string{
						"family": font.Family,
						"style":  font.Style,
					},
				},
			}
			encoder := toml.NewEncoder(f)
			encoder.Indent = ""
			mustHandleError(encoder.Encode(config))
			return
		}
		mustHandleError(err)
		return
	}

	f := mustHaveFile(os.Open(file))
	defer mustHandle(f.Close)

	old := path.Join(path.Dir(file), "alacritty.toml.old")
	c := mustHaveFile(os.Create(old))
	defer mustHandle(c.Close)

	tee := io.TeeReader(f, c)

	var config config
	mustHandle(func() error {
		_, err := toml.NewDecoder(tee).Decode(&config)
		return err
	})

	for k, v := range config {
		fmt.Println("k", k)
		for k, v := range v {
			fmt.Println(">", k, "=", v)
		}
	}

	_, ok := config["font"]
	if ok {
		config["font"]["normal"] = map[string]string{
			"family": font.Family,
			"style":  font.Style,
		}
	} else {
		config["font"] = map[string]any{
			"size": 14.0,
			"normal": map[string]string{
				"family": font.Family,
				"style":  font.Style,
			},
		}
	}

	mustHandle(f.Close)

	f = mustHaveFile(os.Create(file))

	encoder := toml.NewEncoder(f)
	encoder.Indent = ""
	mustHandleError(encoder.Encode(config))

}
