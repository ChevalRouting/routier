package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const defaultIdentityPath = "/var/lib/routier/identity_ed25519"

func newFriendsCommand() *cobra.Command {
	var configPath, identityPath string

	cmd := &cobra.Command{
		Use:   "friends",
		Short: "manage friend routers",
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", defaultConfigPath, "path to config file")
	cmd.PersistentFlags().StringVar(&identityPath, "identity", defaultIdentityPath, "path to ed25519 identity key")
	cmd.AddCommand(
		newFriendsListCommand(&configPath),
		newFriendsCheckCommand(&configPath),
		newFriendsConfigCommand(&configPath),
		newFriendsPreviewCommand(&identityPath),
		newFriendsAddCommand(&configPath, &identityPath),
		newFriendsRemoveCommand(&configPath),
		newFriendsUpdateCommand(&configPath),
		newFriendsPairCommand(&configPath, &identityPath),
		newFriendsSyncCommand(&configPath),
		newFriendsInterfacesCommand(&configPath),
		newFriendsWireguardCommand(&configPath),
	)

	return cmd
}

func newFriendsCheckCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "check <name>",
		Short: "force a health check and info refresh against a single friend",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return newFriendsCheckCommandCallback(configPath, cmd, args)
		},
	}
}

func newFriendsConfigCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "config <name>",
		Short: "fetch and print a friend's live config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return newFriendsConfigCommandCallback(configPath, cmd, args)
		},
	}
}

func newFriendsListCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list configured friends",
		Args:  cobra.NoArgs,
		RunE: func(unusedArg1 *cobra.Command, unusedArg2 []string) error {
			return newFriendsListCommandCallback(configPath, unusedArg1, unusedArg2)
		},
	}
}

func newFriendsPreviewCommand(identityPath *string) *cobra.Command {
	var url, token string
	var tlsSkip bool

	cmd := &cobra.Command{
		Use:   "preview",
		Short: "contact a prospective friend and show its identity",
		Args:  cobra.NoArgs,
		RunE: func(unusedArg4 *cobra.Command, unusedArg5 []string) error {
			return newFriendsPreviewCommandCallback(identityPath, url, token, tlsSkip, unusedArg4, unusedArg5)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "friend base URL")
	cmd.Flags().StringVar(&token, "token", "", "shared bearer token")
	cmd.Flags().BoolVar(&tlsSkip, "tls-skip", false, "skip TLS verification")

	return cmd
}

func newFriendsAddCommand(configPath, identityPath *string) *cobra.Command {
	var name, url, token string
	var tlsSkip bool

	cmd := &cobra.Command{
		Use:   "add",
		Short: "register a friend (always succeeds; verify + pair afterwards)",
		Args:  cobra.NoArgs,
		RunE: func(unusedArg6 *cobra.Command, unusedArg7 []string) error {
			return newFriendsAddCommandCallback(configPath, identityPath, name, url, token, tlsSkip, unusedArg6, unusedArg7)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "friend name")
	cmd.Flags().StringVar(&url, "url", "", "friend base URL")
	cmd.Flags().StringVar(&token, "token", "", "shared bearer token")
	cmd.Flags().BoolVar(&tlsSkip, "tls-skip", false, "skip TLS verification")

	return cmd
}

func newFriendsRemoveCommand(configPath *string) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "remove a friend",
		Args:  cobra.ExactArgs(1),
		RunE: func(unusedArg2 *cobra.Command, args []string) error {
			return newFriendsRemoveCommandCallback(configPath, yes, unusedArg2, args)
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "confirm removal")

	return cmd
}

func newFriendsSyncCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "sync [name]",
		Short: "push HA (vrrp + conntrackd) config to friends and apply locally",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return newFriendsSyncCommandCallback(configPath, cmd, args)
		},
	}
}

func newFriendsInterfacesCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "interfaces <name>",
		Short: "list a friend's interfaces and their addresses",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return newFriendsInterfacesCommandCallback(configPath, cmd, args)
		},
	}
}

func newFriendsWireguardCommand(configPath *string) *cobra.Command {
	var subnet, localIface, localAddr, friendIface, friendAddr string
	var localPort, friendPort int

	cmd := &cobra.Command{
		Use:   "wireguard <name>",
		Short: "derive a link-only WireGuard tunnel to a friend",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return newFriendsWireguardCommandCallback(configPath, subnet, localIface, localAddr, friendIface, friendAddr, localPort, friendPort, cmd, args)
		},
	}

	cmd.Flags().StringVar(&subnet, "subnet", "", "subnet for the two tunnel endpoints (e.g. 169.254.50.0/31)")
	cmd.Flags().StringVar(&localIface, "local-iface", "", "local endpoint interface")
	cmd.Flags().StringVar(&localAddr, "local-addr", "", "local endpoint address (auto if interface has one)")
	cmd.Flags().IntVar(&localPort, "local-port", 0, "local listen port (default shared random)")
	cmd.Flags().StringVar(&friendIface, "friend-iface", "", "friend endpoint interface")
	cmd.Flags().StringVar(&friendAddr, "friend-addr", "", "friend endpoint address (auto if interface has one)")
	cmd.Flags().IntVar(&friendPort, "friend-port", 0, "friend listen port (default shared random)")

	return cmd
}

func newFriendsUpdateCommand(configPath *string) *cobra.Command {
	var url, token string
	var tlsSkip, enabled bool

	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "edit a friend's url, token, tls verification or enabled flag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return newFriendsUpdateCommandCallback(configPath, url, token, tlsSkip, enabled, cmd, args)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "friend base URL")
	cmd.Flags().StringVar(&token, "token", "", "shared bearer token")
	cmd.Flags().BoolVar(&tlsSkip, "tls-skip", false, "skip TLS verification")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "whether the friend is active")

	return cmd
}

func newFriendsPairCommand(configPath, identityPath *string) *cobra.Command {
	var expect string

	cmd := &cobra.Command{
		Use:   "pair <name>",
		Short: "validate a friend's fingerprint and complete pairing",
		Args:  cobra.ExactArgs(1),
		RunE: func(unusedArg3 *cobra.Command, args []string) error {
			return newFriendsPairCommandCallback(configPath, identityPath, expect, unusedArg3, args)
		},
	}

	cmd.Flags().StringVar(&expect, "fingerprint", "", "fingerprint you validated out-of-band")

	return cmd
}

func newFriendsCheckCommandCallback(configPath *string, cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	f := friends.Get(cfg, args[0])
	if f == nil {
		return fmt.Errorf("friend %q not found", args[0])
	}

	st := friends.Check(cmd.Context(), f)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "name\t%s\n", st.Name)
	_, _ = fmt.Fprintf(w, "reachable\t%t\n", st.Reachable)
	if st.LastError != "" {
		_, _ = fmt.Fprintf(w, "error\t%s\n", st.LastError)
	}

	_, _ = fmt.Fprintf(w, "rtt\t%dms\n", st.RTTms)
	_, _ = fmt.Fprintf(w, "hostname\t%s\n", st.Hostname)
	_, _ = fmt.Fprintf(w, "version\t%s\n", st.Version)
	_, _ = fmt.Fprintf(w, "fingerprint\t%s\n", st.Fingerprint)
	_, _ = fmt.Fprintf(w, "identity match\t%t\n", st.IdentityMatch)
	_, _ = fmt.Fprintf(w, "conntrackd\t%t\n", st.ConntrackdRunning)
	for _, v := range st.VRRP {
		_, _ = fmt.Fprintf(w, "vrrp %s\t%s\n", v.Instance, v.State)
	}

	_ = w.Flush()

	if st.Reachable {
		c := friends.NewClient(f.URL, f.Token, f.TLSSkipVerify)
		if ifaces, ierr := c.Interfaces(cmd.Context()); ierr == nil {
			_, _ = fmt.Println("interfaces:")
			for _, iface := range ifaces {
				_, _ = fmt.Printf("  %s: %s\n", iface.Name, strings.Join(iface.Addresses, ", "))
			}
		}

		return nil
	}

	return fmt.Errorf("friend %q is not reachable", st.Name)
}

func newFriendsConfigCommandCallback(configPath *string, cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	f := friends.Get(cfg, args[0])
	if f == nil {
		return fmt.Errorf("friend %q not found", args[0])
	}

	id, _ := identity.LoadOrCreate(defaultIdentityPath)
	client := friends.NewClient(f.URL, f.Token, f.TLSSkipVerify)
	client.SetSealOpener(id, f.Identity.PublicKey)
	remote, err := client.Config(cmd.Context())
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(remote)
	if err != nil {
		return err
	}

	_, _ = fmt.Print(string(data))
	return nil
}

func newFriendsListCommandCallback(configPath *string, _ *cobra.Command, _ []string) error {
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	if len(cfg.Friends) == 0 {
		_, _ = fmt.Println("no friends configured")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tHOSTNAME\tURL\tENABLED\tHA\tFINGERPRINT")
	for _, f := range cfg.Friends {
		ha := f.HA != nil && f.HA.Enabled
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%t\t%s\n",
			f.Name, f.Hostname, f.URL, f.IsEnabled(), ha, f.Identity.Fingerprint)
	}

	return w.Flush()
}

func newFriendsPreviewCommandCallback(identityPath *string, url string, token string, tlsSkip bool, _ *cobra.Command, _ []string) error {
	if url == "" || token == "" {
		return fmt.Errorf("--url and --token are required")
	}

	id, err := identity.LoadOrCreate(*identityPath)
	if err != nil {
		return err
	}

	client := friends.NewClient(url, token, tlsSkip)
	hello, err := client.Hello(context.Background())
	if err != nil {
		return err
	}

	_, _ = fmt.Printf("hostname:    %s\n", hello.Hostname)
	_, _ = fmt.Printf("fingerprint: %s\n", hello.IdentityFingerprint)
	_, _ = fmt.Printf("version:     %s\n", hello.Version)
	if identity.FingerprintMatch(id.Fingerprint(), hello.IdentityFingerprint) {
		_, _ = fmt.Println("warning:     this identity is THIS node (cannot friend self)")
	}

	_, _ = fmt.Println("\nconfirm this fingerprint, then run: routier friends add --fingerprint", hello.IdentityFingerprint)

	return nil
}

func newFriendsAddCommandCallback(configPath *string, identityPath *string, name string, url string, token string, tlsSkip bool, _ *cobra.Command, _ []string) error {
	if name == "" || url == "" || token == "" {
		return fmt.Errorf("--name, --url and --token are required")
	}

	id, err := identity.LoadOrCreate(*identityPath)
	if err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	if friends.Index(cfg, name) >= 0 {
		return fmt.Errorf("friend %q already exists", name)
	}

	f := &config.Friend{Name: name, URL: url, Token: token, TLSSkipVerify: tlsSkip}

	client := friends.NewClient(url, token, tlsSkip)
	if hello, herr := client.Hello(context.Background()); herr != nil {
		_, _ = fmt.Printf("warning: could not reach friend now (%v); added unverified\n", herr)
	} else if identity.FingerprintMatch(id.Fingerprint(), hello.IdentityFingerprint) {
		return fmt.Errorf("refusing to add self as a friend")
	} else {
		f.Hostname = hello.Hostname
		f.Identity.Fingerprint = hello.IdentityFingerprint
		f.Identity.PublicKey = hello.IdentityPublicKey
	}

	if err := friends.Add(cfg, f); err != nil {
		return err
	}

	if err := config.Save(*configPath, cfg); err != nil {
		return err
	}

	if f.Identity.Fingerprint != "" {
		_, _ = fmt.Printf("added friend %q\nfingerprint: %s\n", name, f.Identity.Fingerprint)
		_, _ = fmt.Printf("confirm the fingerprint out-of-band, then pair: routier friends pair %s --fingerprint %s\n", name, f.Identity.Fingerprint)
	} else {
		_, _ = fmt.Printf("added friend %q (unverified)\nverify + pair once it is reachable: routier friends pair %s\n", name, name)
	}

	return nil
}

func newFriendsRemoveCommandCallback(configPath *string, yes bool, _ *cobra.Command, args []string) error {
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	if friends.Index(cfg, args[0]) < 0 {
		return fmt.Errorf("friend %q not found", args[0])
	}

	tagged := friends.TaggedSections(cfg, args[0])
	if !yes {
		for _, s := range tagged {
			_, _ = fmt.Printf("would remove %s %q\n", s.Section, s.Key)
		}

		return fmt.Errorf("this removes friend %q and %d tagged section(s); re-run with --yes to confirm", args[0], len(tagged))
	}

	friends.RemoveTagged(cfg, args[0])
	if _, err := friends.Remove(cfg, args[0]); err != nil {
		return err
	}

	if err := config.Save(*configPath, cfg); err != nil {
		return err
	}

	_, _ = fmt.Printf("removed friend %q and %d tagged section(s)\n", args[0], len(tagged))

	return nil
}

func newFriendsSyncCommandCallback(configPath *string, cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadAndValidate(*configPath, true)
	if err != nil {
		return err
	}

	only := ""
	if len(args) == 1 {
		only = args[0]
	}

	results, err := managers.SyncHA(cmd.Context(), cfg, only)
	for _, res := range results {
		if res.Error != "" {
			_, _ = fmt.Printf("%s: error: %s\n", res.Name, res.Error)
		} else {
			_, _ = fmt.Printf("%s: synced\n", res.Name)
		}
	}

	return err
}

func newFriendsInterfacesCommandCallback(configPath *string, cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	f := friends.Get(cfg, args[0])
	if f == nil {
		return fmt.Errorf("friend %q not found", args[0])
	}

	client := friends.NewClient(f.URL, f.Token, f.TLSSkipVerify)
	ifaces, err := client.Interfaces(cmd.Context())
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "INTERFACE\tADDRESSES")
	for _, iface := range ifaces {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", iface.Name, strings.Join(iface.Addresses, ", "))
	}

	return w.Flush()
}

func newFriendsWireguardCommandCallback(configPath *string, subnet string, localIface string, localAddr string, friendIface string, friendAddr string, localPort int, friendPort int, cmd *cobra.Command, args []string) error {
	if subnet == "" || localIface == "" || friendIface == "" {
		return fmt.Errorf("--subnet, --local-iface and --friend-iface are required")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	f := friends.Get(cfg, args[0])
	if f == nil {
		return fmt.Errorf("friend %q not found", args[0])
	}

	resolvedLocal, err := friends.ResolveEndpointAddr(friends.InterfaceAddresses(cfg), localIface, localAddr)
	if err != nil {
		return fmt.Errorf("local endpoint: %w", err)
	}

	client := friends.NewClient(f.URL, f.Token, f.TLSSkipVerify)
	resolvedFriend := friendAddr
	if resolvedFriend == "" {
		remote, err := client.Interfaces(cmd.Context())
		if err != nil {
			return err
		}

		resolvedFriend, err = friends.ResolveEndpointAddr(remote, friendIface, "")
		if err != nil {
			return fmt.Errorf("friend endpoint: %w", err)
		}
	}

	res, err := friends.DeriveWireguard(friends.WGDeriveParams{
		FriendName:    f.Name,
		LocalHostname: cfg.Hostname,
		Subnet:        subnet,
		LocalAddr:     resolvedLocal,
		LocalPort:     localPort,
		FriendAddr:    resolvedFriend,
		FriendPort:    friendPort,
	})
	if err != nil {
		return err
	}

	if cfg.Wireguard == nil {
		cfg.Wireguard = map[string]*config.Wireguard{}
	}

	cfg.Wireguard[res.InterfaceName] = res.Local
	if err := config.Save(*configPath, cfg); err != nil {
		return err
	}

	_, _ = fmt.Printf("derived %s (friend %s)\n", res.InterfaceName, f.Name)

	if err := managers.PushWireguardCounterpart(f, cfg.Version, res.InterfaceName, res.Counterpart); err != nil {
		_, _ = fmt.Printf("warning: counterpart push failed: %v\n", err)
	} else {
		_, _ = fmt.Println("counterpart pushed to friend")
	}

	return nil
}

func newFriendsUpdateCommandCallback(configPath *string, url string, token string, tlsSkip bool, enabled bool, cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	f := friends.Get(cfg, args[0])
	if f == nil {
		return fmt.Errorf("friend %q not found", args[0])
	}

	fl := cmd.Flags()
	if fl.Changed("url") {
		f.URL = url
	}

	if fl.Changed("token") {
		f.Token = token
	}

	if fl.Changed("tls-skip") {
		f.TLSSkipVerify = tlsSkip
	}

	if fl.Changed("enabled") {
		v := enabled
		f.Enabled = &v
	}

	if err := config.Save(*configPath, cfg); err != nil {
		return err
	}

	_, _ = fmt.Printf("updated friend %q\n", args[0])

	return nil
}

func newFriendsPairCommandCallback(configPath *string, identityPath *string, expect string, _ *cobra.Command, args []string) error {
	id, err := identity.LoadOrCreate(*identityPath)
	if err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	f := friends.Get(cfg, args[0])
	if f == nil {
		return fmt.Errorf("friend %q not found", args[0])
	}

	client := friends.NewClient(f.URL, f.Token, f.TLSSkipVerify)
	hello, err := client.Hello(context.Background())
	if err != nil {
		return fmt.Errorf("contact friend: %w", err)
	}

	if identity.FingerprintMatch(id.Fingerprint(), hello.IdentityFingerprint) {
		return fmt.Errorf("refusing to pair with self")
	}

	switch {
	case expect != "":
		if !identity.FingerprintMatch(expect, hello.IdentityFingerprint) {
			return fmt.Errorf("fingerprint mismatch: expected %s, got %s", expect, hello.IdentityFingerprint)
		}
	case f.Identity.Fingerprint == "":
		_, _ = fmt.Printf("hostname:    %s\nfingerprint: %s\n", hello.Hostname, hello.IdentityFingerprint)
		return fmt.Errorf("confirm the fingerprint above, then re-run with --fingerprint <fp>")
	}

	if f.Identity.Fingerprint != "" && !identity.FingerprintMatch(f.Identity.Fingerprint, hello.IdentityFingerprint) {
		return fmt.Errorf("friend fingerprint changed since it was added (possible MITM)")
	}

	resp, err := client.Pair(context.Background(), id, cfg.Hostname, "", 0)
	if err != nil {
		return fmt.Errorf("pair: %w", err)
	}

	if err := friends.PinIdentity(f, resp.IdentityFingerprint, resp.IdentityPublicKey, resp.X25519PublicKey); err != nil {
		return err
	}

	if resp.Hostname != "" {
		f.Hostname = resp.Hostname
	}

	if err := config.Save(*configPath, cfg); err != nil {
		return err
	}

	_, _ = fmt.Printf("paired with %q (%s)\n", f.Name, resp.IdentityFingerprint)

	return nil
}
