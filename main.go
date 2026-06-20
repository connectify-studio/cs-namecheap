// Command cs-namecheap is a Connectify connector for the Namecheap API.
package main

import (
	"os"

	"github.com/connectify-studio/framework/app"
	"github.com/connectify-studio/framework/creds"

	"github.com/connectify-studio/cs-namecheap/internal/commands"
)

// Build metadata, injected via -ldflags by GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	a, err := app.New(app.Config{
		Name:  "namecheap",
		Short: "Connectify connector for the Namecheap API",
		CredSpec: creds.Spec{Fields: []creds.Field{
			{Key: "api_user", Prompt: "Namecheap API user"},
			{Key: "api_key", Prompt: "Namecheap API key", Secret: true},
			{Key: "username", Prompt: "Namecheap username"},
		}},
	})
	if err != nil {
		panic(err)
	}

	a.Root.Version = version
	a.Root.SetVersionTemplate("cs-namecheap {{.Version}} (commit " + commit + ", built " + date + ")\n")

	a.Root.AddCommand(commands.GetDomains())

	os.Exit(a.Execute())
}
