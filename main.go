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
		ProjectName   string
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
	mod os.FileMode = 0744
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

	var data Args
	flag.StringVar(&data.Path, "output", "", "output destination path")
	flag.StringVar(&data.Path, "o", "", "output destination path (shortened)")
	flag.StringVar(&data.ProjectName, "project-name", "", "project name")
	flag.StringVar(&data.ProjectName, "n", "", "project name (shortened)")
	flag.StringVar(&data.PathToConfig, "c", "", "path to config (shortened)")
	flag.StringVar(&data.PathToConfig, "configuration", "", "path to config")
	flag.StringVar(&data.TemplatesPath, "t", "", "path to templates (shortened)")
	flag.StringVar(&data.TemplatesPath, "templates", "", "path to config")
	flag.Parse()

	if err := data.valid(); err != nil {
		fmt.Printf("fatal error: %+v\n", err)
		return
	}

	// import templates
	tmpl, err := template.New("main").ParseFS(os.DirFS(data.TemplatesPath), "*.tmpl")
	if err != nil {
		log.Fatal(fmt.Errorf("failure to create the main.go file: %s", err))
	}

	f, err := os.Open(data.PathToConfig)
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
	// create output dir
	projectRootPath := path.Join(data.Path, data.ProjectName)
	if err = os.MkdirAll(projectRootPath, mod); err != nil {
		fmt.Printf("failure to create cache directory: %s", err)
		return
	}
	for _, file := range cfg.Files {
		if err = processFile(tmpl, projectRootPath, cfg.Global, &file); err != nil {
			fmt.Printf("fatal error: %+v\n", err)
			return
		}
	}
	if len(cfg.Cmds) > 0 {
		for _, c := range cfg.Cmds {
			cmd := exec.CommandContext(ctx, c.Name, c.Args...)
			cmd.Dir = projectRootPath
			if b, err := cmd.CombinedOutput(); err != nil {
				fmt.Printf("Output: %+v\nError: %+v\n", string(b), err)
				return
			}
		}
	}
}

func processFile(tmpl *template.Template, projectRootPath string, global map[string]any, data *File) error {
	dir := fmt.Sprintf("%s/%s", projectRootPath, data.Path)
	if err := os.MkdirAll(dir, mod); err != nil {
		return fmt.Errorf("failure to create %s directory: %s", dir, err)
	}
	file := fmt.Sprintf("%s/%s", dir, data.Name)
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("failure to create %s file: %s", file, err)
	}
	if err = tmpl.ExecuteTemplate(f, data.Template, Values{Global: global, Local: data.Local}); err != nil {
		return fmt.Errorf("failure to populate %s file: %s", data.Name, err)
	}
	return nil
}
