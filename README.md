# gg

simple utility to facilitate mundane work by generating code based on templates and variables.

## Conventions

`gg` hinges on the following:
1. json schema with definitions that will be used for code generation;
2. [text/template](https://pkg.go.dev/text/template) package and its semantics;
3. template files should end with `.tmpl`.

## Requirements

The programme needs **Go ^1.18** to be compiled.

## Installation

    go install github.com/omerkaya1/gg@latest

## Usage
    Usage of gg:
        -c string 
            path to config (shortened)
        -configuration string
            path to config
        -o string
            output destination path (shortened)
        -output string
            output destination path
        -s
            print files to STDOUT and prepend them with a file name (shortened)
        -separate
            print files to STDOUT and prepend them with a file name
        -t string
            path to templates (shortened)
        -templates string
            path to config

## Config file semantics

Currently, `gg` supports global template variables (the ones that span throughout files and may occur in a lot of places)
and local variables (the ones that are specific to the file and won't be used elsewhere).

Placeholders for global variables should be prefixed with _.Global_ pattern, while local ones with _.Local_.


## Example

By default, `gg` prints its output to STDOUT.

If the template path is not provided, `gg` looks for templates in the working directory it was called.

Otherwise, use `-o path` to specify output directory.

***

Suppose we have this setup:

templates are stored in our local directory:

```
    $ ls -a ./templates
    main.tmpl              go.mod.tmpl
```

_main.go file template_
```gotemplate
package main

import (
    "fmt"
)

const (
	serviceName = "{{.Global.ServiceName}}"
)

func main() { {{if .Local.PrintThis}}
    fmt.Printf("this is: "){{end}}
    fmt.Println(serviceName)
}
```

_go.mod file template_
```gotemplate
module {{.Global.ModuleName}}

go {{.Local.GoVersion}}
```

_json config file_
```json
{
  "global": {
    "ServiceName": "New Awesome Service",
    "ModuleName": "github.com/omerkaya1/new-awesome-service"
  },
  "files": [
    {
      "name": "main.go",
      "template": "main.tmpl",
      "path": "",
      "local": {
        "PrintThis": true
      }
    },
    {
      "name": "go.mod",
      "template": "go.mod.tmpl",
      "path": "",
      "local": {
        "GoVersion": "1.19"
      }
    }
  ],
  "commands": [
    {
      "name": "gofmt",
      "args": [
        "-s"
      ]
    },
    {
      "name": "go",
      "args": [
        "mod",
        "tidy",
        "-e"
      ]
    }
  ]
}
```

Running:

    gg -c config.json -o /destination/path -t /path/to/templates

Produces:

```
$ ls /destination/path
go.mod  main.go

```

With:

- /destination/path/main.go
```
package main

import (
    "fmt"
)

const (
	serviceName = "New Awesome Service"
)

func main() { 
    fmt.Printf("this is: ")
    fmt.Println(serviceName)
}
```

- /destination/path/go.mod
```
module github.com/omerkaya1/new-awesome-service

go 1.19
```

See [examples](examples) for more details.

You can achieve the same result by just piping config data to the programme:

```shell
cat examples/config.json | gg -t examples
```
or
```shell
gg -t examples < examples/config.json
```

## Uninstall

Simple deletion of the binary should suffice:
```shell
rm $(which gg)
```