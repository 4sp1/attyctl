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
	"strings"

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
	flag.Parse()

	if !*setupFont {
		fmt.Println("Nothing to do. Bye.")
	}

	if *setupFontFamily == "" {
		cmd := exec.Command("fzf")
		cmd.Stdin = setupFontCache(*setupFlag, *setupJson)
		var b bytes.Buffer
		cmd.Stdout = &b
		cmd.Stderr = os.Stderr
		mustHandle(cmd.Run)
		*setupFontFamily = strings.TrimSpace(b.String())
	}

	switch {
	case *setupAlacritty:
		alacritty(*setupFontFamily)
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

func setupFontCache(force bool, spFontsJsonFile string) io.Reader {
	home := mustHaveString(os.UserHomeDir())
	file := path.Join(home, ".local", "share")
	mustHandleError(os.MkdirAll(file, 0700))
	file = path.Join(file, "fonts.txt")

	var create bool
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			create = true
		} else {
			mustHandleError(err)
		}
	}

	if !create && !force {
		f := mustHaveFile(os.Open(file))
		defer mustHandle(f.Close)
		var b bytes.Buffer
		mustHandle(func() error {
			_, err := io.Copy(&b, f)
			return err
		})
		return &b
	}

	var b, w bytes.Buffer

	cmd := exec.Command("system_profiler", "-json", "SPFontsDataType")
	cmd.Stdout = &b
	fmt.Fprintln(os.Stderr, "Creating the font cache—please be patient, this could take some time.")
	mustHandle(cmd.Run)

	var r io.Reader
	r = &b
	if len(spFontsJsonFile) > 0 {
		log := mustHaveFile(os.Create("sp_fonts_data_type.json"))
		defer mustHandle(log.Close)
		r = io.TeeReader(&b, log)
	}

	var spf spFontsDataTypeObjectJSON
	mustHandleError(json.NewDecoder(r).Decode(&spf))

	families := make(map[string]struct{})
	for _, spf := range spf.SPFontsDataType {
		for _, tf := range spf.Typefaces {
			families[tf.Family] = struct{}{}
		}
	}

	f := mustHaveFile(os.Create(file))

	for family := range families {
		mustHandle(func() error {
			_, err := io.MultiWriter(&w, f).Write([]byte(family + "\n"))
			return err
		})
	}

	return &w
}

func terminal(font string) error {
	cmd := exec.Command("osascript", "-e",
		`tell application "Terminal" to set the font name of window 1 to "`+font+`"`)
	return cmd.Run()
}

func alacritty(font string) {
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
						"family": font,
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
		config["font"]["normal"] = map[string]string{"family": font}
	} else {
		config["font"] = map[string]any{
			"size": 14.0,
			"normal": map[string]string{
				"family": font,
			},
		}
	}

	mustHandle(f.Close)

	f = mustHaveFile(os.Create(file))

	encoder := toml.NewEncoder(f)
	encoder.Indent = ""
	mustHandleError(encoder.Encode(config))

}

type spFontsDataTypeObjectJSON struct {
	SPFontsDataType []spFontsDataTypeJSON
}
type spFontsDataTypeJSON struct {
	Typefaces []spTypeface `json:"typefaces"`
}
type spTypeface struct {
	Family string `json:"family"`
}
