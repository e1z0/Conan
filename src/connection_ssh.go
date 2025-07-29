package main

import (
	"bytes"
	"html/template"
	"log"
)

func CmdParseTemplate(input string) string {
	cmdtpl := struct {
		Home      string
		AppDir    string
		ConfigDir string
	}{
		Home:      env.homeDir,
		AppDir:    env.appPath,
		ConfigDir: env.configDir,
	}

	defaultOutput := func(tpl string) string {
		tmpl, err := template.New("cmdtpl").
			Option("missingkey=default").
			Parse(tpl)
		if err != nil {
			log.Printf("Invalid cmdline template (parse): %v — using literal", err)
			return tpl
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, cmdtpl); err != nil {
			log.Printf("Invalid cmdline template (exec): %v — using literal", err)
			return tpl
		}

		return buf.String()
	}
	return defaultOutput(input)
}
