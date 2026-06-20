// Package commands implements the cs-namecheap subcommands.
package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/connectify-studio/framework/app"
	"github.com/connectify-studio/framework/httpx"
	"github.com/connectify-studio/framework/render"

	"github.com/connectify-studio/cs-namecheap/internal/namecheap"
)

// GetDomains returns the `get-domains` command.
func GetDomains() *cobra.Command {
	var (
		page     int
		pageSize int
		search   string
		sandbox  bool
		clientIP string
	)
	cmd := &cobra.Command{
		Use:   "get-domains",
		Short: "List domains in the account",
		Long: "List domains via namecheap.domains.getList.\n\n" +
			"Requires API access enabled on your Namecheap account and the client IP " +
			"whitelisted in the dashboard. On first run you will be prompted for your " +
			"API user, API key, and username.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app.FromContext(cmd.Context())
			ctx := cmd.Context()

			creds, err := resolveCredentials(ctx, a, clientIP)
			if err != nil {
				return err
			}

			hc := httpx.New(httpx.WithLogger(a.Log.Logger), httpx.WithTimeout(30*time.Second))
			client := namecheap.New(hc, creds, sandbox)

			a.Log.Info("fetching domains", "page", page, "page_size", pageSize, "search", search, "sandbox", sandbox)
			opts := namecheap.GetListOptions{
				Page:       page,
				PageSize:   pageSize,
				SearchTerm: search,
			}
			// Without an explicit --page, return every domain by paging through
			// the full result set; with --page, honor that single page.
			var domains []namecheap.Domain
			if page > 0 {
				domains, err = client.GetDomains(ctx, opts)
			} else {
				domains, err = client.GetAllDomains(ctx, opts)
			}
			if err != nil {
				return err
			}

			res := render.Result{Columns: []string{"Name", "Created", "Expires", "AutoRenew", "WhoisGuard", "IsLocked"}}
			for _, d := range domains {
				res.Rows = append(res.Rows, []string{d.Name, d.Created, d.Expires, d.AutoRenew, d.WhoisGuard, d.IsLocked})
			}
			return a.Render(res)
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "fetch only this page (1-based); default fetches all pages")
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "results per page (max 100)")
	cmd.Flags().StringVar(&search, "search", "", "filter by search term")
	cmd.Flags().BoolVar(&sandbox, "sandbox", false, "use the Namecheap sandbox API")
	cmd.Flags().StringVar(&clientIP, "client-ip", "", "client IP to send (default: auto-detected public IP)")
	return cmd
}

// resolveCredentials reads the stored/prompted API credentials and resolves the
// ClientIp, auto-detecting the public IP when not supplied.
func resolveCredentials(ctx context.Context, a *app.App, clientIP string) (namecheap.Credentials, error) {
	apiUser, err := a.Creds.Require("api_user")
	if err != nil {
		return namecheap.Credentials{}, err
	}
	apiKey, err := a.Creds.Require("api_key")
	if err != nil {
		return namecheap.Credentials{}, err
	}
	userName, err := a.Creds.Require("username")
	if err != nil {
		return namecheap.Credentials{}, err
	}

	ip := clientIP
	if ip == "" {
		// Cache the detected IP in state to avoid repeat lookups.
		if cached, ok := a.State.Get("client_ip"); ok {
			ip = cached
		} else {
			if ip, err = detectPublicIP(ctx); err != nil {
				return namecheap.Credentials{}, fmt.Errorf("detecting public IP (pass --client-ip to override): %w", err)
			}
			_ = a.State.Set("client_ip", ip)
		}
	}

	return namecheap.Credentials{
		APIUser:  apiUser,
		APIKey:   apiKey,
		UserName: userName,
		ClientIP: ip,
	}, nil
}

// detectPublicIP queries a public echo service for the caller's outbound IP,
// which must match a Namecheap-whitelisted address.
func detectPublicIP(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "", fmt.Errorf("empty response from IP service")
	}
	return ip, nil
}
