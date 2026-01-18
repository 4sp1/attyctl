package fonts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/4sp1/must"
)

func NewCache(opts ...Option) Cache {
	c := defaultConf()
	for _, opt := range opts {
		c = opt(c)
	}
	return cache{
		mustHandle:      must.Handle(c.mustErrorHandler),
		mustHandleError: must.HandleError(c.mustErrorHandler),
		mustHaveString:  must.Have(c.mustStringController),
		mustHaveFile:    must.Have(c.mustFileController),
	}
}

type Cache interface {
	NewReader() io.Reader
	Refresh() io.Reader
}

type Option func(Conf) Conf

func WithWriteSystemProfilerFontsTo(file string) Option {
	return func(c Conf) Conf {
		c.spFontsFile = file
		return c
	}
}

type Conf struct {
	spFontsFile string

	mustErrorHandler     must.ErrorHandler
	mustStringController must.Controller[string]
	mustFileController   must.Controller[*os.File]
}

func defaultConf() Conf {
	var c Conf
	c.mustErrorHandler = must.ExitHandler(1)
	c.mustFileController = must.ExitController[*os.File](1)
	c.mustStringController = must.ExitController[string](1)
	return c
}

type cache struct {
	config Conf

	mustHandle      must.ErrorHandlerFn
	mustHandleError must.ErrorValueHandlerFn
	mustHaveString  must.ControllerHaveFn[string]
	mustHaveFile    must.ControllerHaveFn[*os.File]
}

func (c cache) NewReader() io.Reader {
	return c.setupFontCache(false, c.config.spFontsFile)
}

func (c cache) Refresh() io.Reader {
	return c.setupFontCache(true, c.config.spFontsFile)
}

func (c cache) setupFontCache(force bool, spFontsJsonFile string) io.Reader {
	home := c.mustHaveString(os.UserHomeDir())
	file := path.Join(home, ".local", "share")
	c.mustHandleError(os.MkdirAll(file, 0700))
	file = path.Join(file, "fonts.txt")

	var create bool
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			create = true
		} else {
			c.mustHandleError(err)
		}
	}

	if !create && !force {
		f := c.mustHaveFile(os.Open(file))
		defer c.mustHandle(f.Close)
		var b bytes.Buffer
		c.mustHandle(func() error {
			_, err := io.Copy(&b, f)
			return err
		})
		return &b
	}

	var b, w bytes.Buffer

	cmd := exec.Command("system_profiler", "-json", "SPFontsDataType")
	cmd.Stdout = &b
	fmt.Fprintln(os.Stderr, "Creating the font cache—please be patient, this could take some time.")
	c.mustHandle(cmd.Run)

	var r io.Reader
	r = &b
	if len(spFontsJsonFile) > 0 {
		log := c.mustHaveFile(os.Create(spFontsJsonFile))
		defer c.mustHandle(log.Close)
		r = io.TeeReader(&b, log)
	}

	var spf spFontsDataTypeObjectJSON
	c.mustHandleError(json.NewDecoder(r).Decode(&spf))

	families := make(map[string]map[string]struct{})
	for _, spf := range spf.SPFontsDataType {
		for _, tf := range spf.Typefaces {
			if _, exists := families[tf.Family]; !exists {
				families[tf.Family] = make(map[string]struct{})
			}
			families[tf.Family][tf.Style] = struct{}{}
		}
	}

	f := c.mustHaveFile(os.Create(file))

	for family, styles := range families {
		for style := range styles {
			m := io.MultiWriter(&w, f)
			c.mustHandleError(
				json.NewEncoder(m).Encode(Font{
					Family: family,
					Style:  style,
				}))
		}
	}

	return &w
}
