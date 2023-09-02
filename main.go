package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"text/template"
)

type (
	Args struct {
		Separator     bool
		PathToConfig  string
		TemplatesPath string
		OutputPath    string
	}
	Config struct {
		Global map[string]any `json:"global"`
		Files  []File         `json:"files,omitempty"`
		Cmds   []Command      `json:"commands,omitempty"`
	}
	File struct {
		Name     string         `json:"name"`
		Path     string         `json:"path"`
		Template string         `json:"template"`
		Local    map[string]any `json:"local"`
	}
	Command struct {
		Name string   `json:"name"`
		Args []string `json:"args"`
	}
	Values struct {
		Global map[string]any
		Local  map[string]any
	}
)

var (
	args Args
	mod  os.FileMode = 0744
)

func (a *Args) valid() error {
	if a.Separator && a.OutputPath != "" {
		return fmt.Errorf("wrong argumnets: cannot use STDOUT and output dir for file generation")
	}
	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	flag.StringVar(&args.OutputPath, "output", "", "output destination path")
	flag.StringVar(&args.OutputPath, "o", "", "output destination path (shortened)")
	flag.StringVar(&args.PathToConfig, "configuration", "", "path to config")
	flag.StringVar(&args.PathToConfig, "c", "", "path to config (shortened)")
	flag.StringVar(&args.TemplatesPath, "templates", "", "path to templates")
	flag.StringVar(&args.TemplatesPath, "t", "", "path to templates (shortened)")
	flag.Parse()

	if err := args.valid(); err != nil {
		log.Fatalf("fatal error: %+v\n", err)
	}

	var (
		f   = os.Stdin
		err error
	)
	if args.PathToConfig == "" {
		stat, err := f.Stat()
		if err != nil {
			log.Fatalf("fatal error: %+v\n", err)
		}
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			log.Fatalln("fatal error: no input from stdin")
		}
	} else {
		if f, err = os.Open(args.PathToConfig); err != nil {
			log.Fatalf("fatal error: %+v\n", err)
		}
		defer f.Close()
	}

	// read in config
	var cfg Config
	if err = json.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatalf("fatal error: %+v\n", err)
	}

	// empty template path, assume that templates are inside the working directory
	if args.TemplatesPath == "" {
		if args.TemplatesPath, err = os.Getwd(); err != nil {
			log.Fatalf("fatal error: %+v\n", err)
		}
	}

	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
		"ToLower": strings.ToLower,
		"ToTitle": strings.ToTitle,
	}

	// import templates
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(os.DirFS(args.TemplatesPath), "*.tmpl")
	if err != nil {
		log.Fatal(fmt.Errorf("failure to initialise template: %s", err))
	}

	// create an output dir if necessary
	if args.OutputPath != "" {
		if err = os.MkdirAll(args.OutputPath, mod); err != nil {
			log.Fatalf("failure to create output directory: %s", err)
		}
	}

	// iterate over specified files
	for _, file := range cfg.Files {
		if err = processFile(tmpl, cfg.Global, &file); err != nil {
			log.Fatalf("fatal error: %+v\n", err)
		}
	}

	// run commands
	if len(cfg.Cmds) > 0 {
		for _, c := range cfg.Cmds {
			cmd := exec.CommandContext(ctx, c.Name, c.Args...)
			cmd.Dir = args.OutputPath
			if b, err := cmd.CombinedOutput(); err != nil {
				log.Fatalf("Output: %+v\nError: %+v\n", string(b), err)
			}
		}
	}
}

const separator = "---%s"

func processFile(tmpl *template.Template, global map[string]any, file *File) error {
	var err error
	if args.OutputPath == "" {
		if args.Separator {
			_, _ = fmt.Fprintf(os.Stdout, separator, file.Name)
		}
		if err = tmpl.ExecuteTemplate(os.Stdout, file.Template, Values{Global: global, Local: file.Local}); err != nil {
			return fmt.Errorf("failure to populate %s file: %s", file.Name, err)
		}
		return nil
	}
	if err = os.MkdirAll(path.Join(args.OutputPath, file.Path), mod); err != nil {
		return fmt.Errorf("failure to create %s directory: %s", path.Join(args.OutputPath, file.Path), err)
	}
	f, err := os.Create(path.Join(args.OutputPath, file.Path, file.Name))
	if err != nil {
		return fmt.Errorf("failure to create %s file: %s", file, err)
	}
	if err = tmpl.ExecuteTemplate(f, file.Template, Values{Global: global, Local: file.Local}); err != nil {
		return fmt.Errorf("failure to populate %s file: %s", file.Name, err)
	}
	return nil
}
