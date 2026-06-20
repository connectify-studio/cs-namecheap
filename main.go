// Command cs-namecheap is a Connectify connector for the Namecheap API.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

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

// credFields declares the only credentials this connector recognizes. It seeds
// both the credential spec (prompting, masking, env lookup) and the allow-list
// enforced on `config set`.
var credFields = []creds.Field{
	{Key: "api_user", Prompt: "Namecheap API user"},
	{Key: "api_key", Prompt: "Namecheap API key", Secret: true},
	{Key: "username", Prompt: "Namecheap username"},
}

func main() {
	a, err := app.New(app.Config{
		Name:     "namecheap",
		Short:    "Connectify connector for the Namecheap API",
		CredSpec: creds.Spec{Fields: credFields},
	})
	if err != nil {
		panic(err)
	}

	a.Root.Version = version
	a.Root.SetVersionTemplate("cs-namecheap {{.Version}} (commit " + commit + ", built " + date + ")\n")

	restrictConfigKeys(a, credFields)
	a.Root.AddCommand(commands.GetDomains())

	os.Exit(a.Execute())
}

// restrictConfigKeys hardens the framework-provided `config set` so it only
// accepts the credentials this connector actually uses. Without this, any key
// (e.g. a typo'd "auth_user") could be written to credentials.json where it
// would silently be ignored.
func restrictConfigKeys(a *app.App, fields []creds.Field) {
	allowed := make(map[string]bool, len(fields))
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		allowed[f.Key] = true
		names = append(names, f.Key)
	}

	cfg, _, err := a.Root.Find([]string{"config"})
	if err != nil || cfg == nil {
		return
	}
	for _, sub := range cfg.Commands() {
		if sub.Name() != "set" {
			continue
		}
		sub.RunE = func(_ *cobra.Command, args []string) error {
			if !allowed[args[0]] {
				return fmt.Errorf("unknown credential %q; valid keys are: %s", args[0], strings.Join(names, ", "))
			}
			return a.Creds.Set(args[0], args[1])
		}
	}
}
