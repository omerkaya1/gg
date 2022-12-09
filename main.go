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
	"syscall"
	"text/template"
)

type (
	Args struct {
		PathToConfig  string
		TemplatesPath string
		Path          string
	}
	Config struct {
		Global map[string]any `json:"global"`
		Files  []File         `json:"files,omitempty"`
		Cmds   []Commands     `json:"commands,omitempty"`
	}
	File struct {
		Name     string         `json:"name"`
		Path     string         `json:"path"`
		Template string         `json:"template"`
		Local    map[string]any `json:"local"`
	}
	Commands struct {
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
	stat, err := os.Stat(a.Path)
	if err != nil {
		return fmt.Errorf("incorrect path: %s", err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", a.Path)
	}
	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	flag.StringVar(&args.Path, "output", "", "output destination path")
	flag.StringVar(&args.Path, "o", "", "output destination path (shortened)")
	flag.StringVar(&args.PathToConfig, "c", "", "path to config (shortened)")
	flag.StringVar(&args.PathToConfig, "configuration", "", "path to config")
	flag.StringVar(&args.TemplatesPath, "t", "", "path to templates (shortened)")
	flag.StringVar(&args.TemplatesPath, "templates", "", "path to config")
	flag.Parse()

	if err := args.valid(); err != nil {
		fmt.Printf("fatal error: %+v\n", err)
		return
	}

	// import templates
	tmpl, err := template.New("main").ParseFS(os.DirFS(args.TemplatesPath), "*.tmpl")
	if err != nil {
		log.Fatal(fmt.Errorf("failure to create the main.go file: %s", err))
	}

	f, err := os.Open(args.PathToConfig)
	if err != nil {
		fmt.Printf("fatal error: %+v\n", err)
		return
	}
	defer f.Close()

	var cfg Config
	if err = json.NewDecoder(f).Decode(&cfg); err != nil {
		fmt.Printf("fatal error: %+v\n", err)
		return
	}
	for _, file := range cfg.Files {
		if err = processFile(tmpl, cfg.Global, &file); err != nil {
			fmt.Printf("fatal error: %+v\n", err)
			return
		}
	}
	if len(cfg.Cmds) > 0 {
		for _, c := range cfg.Cmds {
			cmd := exec.CommandContext(ctx, c.Name, c.Args...)
			cmd.Dir = args.Path
			if b, err := cmd.CombinedOutput(); err != nil {
				fmt.Printf("Output: %+v\nError: %+v\n", string(b), err)
				return
			}
		}
	}
}

func processFile(tmpl *template.Template, global map[string]any, file *File) error {
	if err := os.MkdirAll(path.Join(args.Path, file.Path), mod); err != nil {
		return fmt.Errorf("failure to create %s directory: %s", path.Join(args.Path, file.Path), err)
	}
	f, err := os.Create(path.Join(args.Path, file.Path, file.Name))
	if err != nil {
		return fmt.Errorf("failure to create %s file: %s", file, err)
	}
	if err = tmpl.ExecuteTemplate(f, file.Template, Values{Global: global, Local: file.Local}); err != nil {
		return fmt.Errorf("failure to populate %s file: %s", file.Name, err)
	}
	return nil
}
