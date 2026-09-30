package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	instances map[string]*instance
	mcp       *mcp.Server
}

type noInput struct{}

type instanceInput struct {
	Instance string `json:"instance" jsonschema:"name of the configured Routier instance"`
}

type sessionInput struct {
	Instance string `json:"instance" jsonschema:"name of the configured Routier instance"`
	Session  string `json:"session" jsonschema:"Routier configuration session ID"`
}

type sessionYAMLInput struct {
	Instance string `json:"instance" jsonschema:"name of the configured Routier instance"`
	Session  string `json:"session" jsonschema:"Routier configuration session ID"`
	YAML     string `json:"yaml" jsonschema:"complete canonical Routier YAML configuration"`
}

type historyInput struct {
	Instance string `json:"instance" jsonschema:"name of the configured Routier instance"`
	Minutes  int    `json:"minutes,omitempty" jsonschema:"lookback in minutes, from 1 to 43200"`
	Iface    string `json:"iface,omitempty" jsonschema:"optional interface filter"`
	Series   string `json:"series,omitempty" jsonschema:"optional comma-separated series: interfaces,bgp,proto,system,total,usage,probes"`
}

type privateKeyInput struct {
	Instance   string `json:"instance" jsonschema:"name of the configured Routier instance"`
	PrivateKey string `json:"private_key" jsonschema:"WireGuard private key"`
}

type execInput struct {
	Instance string `json:"instance" jsonschema:"name of the configured Routier instance"`
	Command  string `json:"command" jsonschema:"shell command to run in the instance's debug console"`
	Timeout  int    `json:"timeout,omitempty" jsonschema:"maximum seconds to wait for the command, 1 to 600, default 30"`
}

type toolOutput struct {
	Instance string `json:"instance,omitempty"`
	Result   any    `json:"result"`
}

func New(configPath, version string) (*Server, error) {
	instances, err := loadInstances(configPath)
	if err != nil {
		return nil, err
	}

	s := &Server{instances: instances}
	s.mcp = mcp.NewServer(&mcp.Implementation{Name: "routier", Version: version}, nil)
	s.registerTools()
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	return s.mcp.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) instance(name string) (*instance, error) {
	i, ok := s.instances[name]
	if !ok {
		return nil, fmt.Errorf("unknown instance %q", name)
	}

	return i, nil
}

func (s *Server) call(ctx context.Context, in instanceInput, method, path string) (*mcp.CallToolResult, any, error) {
	i, err := s.instance(in.Instance)
	if err != nil {
		return nil, toolOutput{}, err
	}

	result, err := i.request(ctx, method, path, "", nil)
	return nil, toolOutput{Instance: in.Instance, Result: result}, err
}

func (s *Server) registerTools() {
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "instances_list", Description: "List configured Routier instances without exposing credentials."}, s.instancesList)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "instance_status", Description: "Check an instance and return its canonical config metadata and pending apply state."}, s.instanceStatus)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "instance_interfaces", Description: "List an instance's live network interfaces with their real kernel addresses, including ones assigned by DHCPv4, DHCPv6, or SLAAC. Use this to determine an instance's actual IP addresses rather than its configured intent."}, s.instanceInterfaces)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "config_get", Description: "Read the committed canonical Routier YAML and its SHA-256 hash."}, s.configGet)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "config_schema", Description: "Read the JSON Schema for the canonical Routier configuration, generated from the running instance's own types. Use this to learn the exact structure, field names, and required keys before writing YAML with config_session_put_yaml, instead of guessing the schema."}, s.configSchema)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "config_session_create", Description: "Create an isolated configuration session from the current committed config."}, s.sessionCreate)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "config_session_get", Description: "Read the complete canonical YAML held by a configuration session."}, s.sessionGet)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "config_session_put_yaml", Description: "Replace a configuration session with a complete canonical Routier YAML document. This does not apply it."}, s.sessionPutYAML)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "config_session_diff", Description: "Return a session's YAML diff against the committed configuration."}, s.sessionDiff)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "config_session_validate", Description: "Fully validate a configuration session without applying it."}, s.sessionValidate)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "config_session_apply", Description: "Apply a previously reviewed configuration session and arm Routier's rollback watchdog."}, s.sessionApply)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "config_session_discard", Description: "Discard a configuration session without changing live configuration."}, s.sessionDiscard)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "apply_pending", Description: "Read the pending apply and rollback-watchdog state."}, s.applyPending)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "apply_confirm", Description: "Confirm a pending apply, keeping the new configuration."}, s.applyConfirm)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "apply_rollback", Description: "Restore the snapshot associated with the pending apply."}, s.applyRollback)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "stats_current", Description: "Read current system, interface, BGP, and OSPF statistics."}, s.statsCurrent)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "stats_history", Description: "Read filtered historical Routier statistics."}, s.statsHistory)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "stats_neighbors", Description: "Read the latest stored neighbor statistics."}, s.statsNeighbors)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "stats_lldp", Description: "Read the latest stored LLDP/CDP link-layer neighbors."}, s.statsLLDP)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "stats_processes", Description: "Read the live process list."}, s.statsProcesses)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "debug_exec", Description: "Run a shell command on an instance through its debug console (the same root websocket terminal the web UI exposes) and return the command's output and exit code. This grants full shell access; use it deliberately."}, s.debugExec)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "wireguard_keypair_generate", Description: "Generate a WireGuard keypair on a Routier instance."}, s.wireguardKeypair)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: "wireguard_public_key_derive", Description: "Derive a WireGuard public key from a private key."}, s.wireguardPublicKey)
}

func (s *Server) instancesList(_ context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, any, error) {
	names := make([]string, 0, len(s.instances))
	for name := range s.instances {
		names = append(names, name)
	}

	sort.Strings(names)
	return nil, toolOutput{Result: names}, nil
}

func (s *Server) instanceStatus(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	i, err := s.instance(in.Instance)
	if err != nil {
		return nil, toolOutput{}, err
	}

	configResult, err := i.request(ctx, http.MethodGet, "/api/v1/config", "", nil)
	if err != nil {
		return nil, toolOutput{}, err
	}

	pending, err := i.request(ctx, http.MethodGet, "/api/apply/pending", "", nil)
	if err != nil {
		return nil, toolOutput{}, err
	}

	return nil, toolOutput{Instance: in.Instance, Result: map[string]any{"config": configResult, "apply": pending}}, nil
}

func (s *Server) instanceInterfaces(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, in, http.MethodGet, "/api/system/nics")
}

func (s *Server) configGet(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, in, http.MethodGet, "/api/v1/config")
}

func (s *Server) configSchema(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	i, err := s.instance(in.Instance)
	if err != nil {
		return nil, toolOutput{}, err
	}

	version, err := i.schemaVersion(ctx)
	if err == nil && version == cfgpkg.CurrentVersion {
		doc, err := embeddedSchema()
		if err == nil {
			return nil, toolOutput{Instance: in.Instance, Result: doc}, nil
		}
	}

	return s.call(ctx, in, http.MethodGet, "/api/config/schema")
}

func (s *Server) sessionCreate(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, in, http.MethodPost, "/api/v1/sessions")
}

func (s *Server) sessionPath(in sessionInput, suffix string) string {
	return "/api/v1/sessions/" + url.PathEscape(in.Session) + suffix
}

func (s *Server) sessionGet(ctx context.Context, _ *mcp.CallToolRequest, in sessionInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, instanceInput{Instance: in.Instance}, http.MethodGet, s.sessionPath(in, "/config"))
}

func (s *Server) sessionPutYAML(ctx context.Context, _ *mcp.CallToolRequest, in sessionYAMLInput) (*mcp.CallToolResult, any, error) {
	i, err := s.instance(in.Instance)
	if err != nil {
		return nil, toolOutput{}, err
	}

	path := s.sessionPath(sessionInput{Instance: in.Instance, Session: in.Session}, "/config")
	result, err := i.request(ctx, http.MethodPut, path, "application/yaml", []byte(in.YAML))
	return nil, toolOutput{Instance: in.Instance, Result: result}, err
}

func (s *Server) sessionDiff(ctx context.Context, _ *mcp.CallToolRequest, in sessionInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, instanceInput{Instance: in.Instance}, http.MethodGet, s.sessionPath(in, "/diff"))
}

func (s *Server) sessionValidate(ctx context.Context, _ *mcp.CallToolRequest, in sessionInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, instanceInput{Instance: in.Instance}, http.MethodPost, s.sessionPath(in, "/validate"))
}

func (s *Server) sessionApply(ctx context.Context, _ *mcp.CallToolRequest, in sessionInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, instanceInput{Instance: in.Instance}, http.MethodPost, s.sessionPath(in, "/apply"))
}

func (s *Server) sessionDiscard(ctx context.Context, _ *mcp.CallToolRequest, in sessionInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, instanceInput{Instance: in.Instance}, http.MethodDelete, s.sessionPath(in, ""))
}

func (s *Server) applyPending(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, in, http.MethodGet, "/api/apply/pending")
}

func (s *Server) applyConfirm(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, in, http.MethodPost, "/api/apply/confirm")
}

func (s *Server) applyRollback(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	i, err := s.instance(in.Instance)
	if err != nil {
		return nil, toolOutput{}, err
	}

	pending, err := i.request(ctx, http.MethodGet, "/api/apply/pending", "", nil)
	if err != nil {
		return nil, toolOutput{}, err
	}

	p, ok := pending.(map[string]any)
	if !ok || p["pending"] != true {
		return nil, toolOutput{}, fmt.Errorf("%s: no pending apply", in.Instance)
	}

	snapID, _ := p["snap_id"].(string)
	if snapID == "" {
		return nil, toolOutput{}, fmt.Errorf("%s: pending apply has no snapshot id", in.Instance)
	}

	result, err := i.request(ctx, http.MethodPost, "/api/snapshots/"+url.PathEscape(snapID)+"/restore", "", nil)
	return nil, toolOutput{Instance: in.Instance, Result: result}, err
}

func (s *Server) statsCurrent(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, in, http.MethodGet, "/api/stats")
}

func (s *Server) statsHistory(ctx context.Context, _ *mcp.CallToolRequest, in historyInput) (*mcp.CallToolResult, any, error) {
	query := url.Values{}
	if in.Minutes > 0 {
		query.Set("minutes", strconv.Itoa(in.Minutes))
	}

	if in.Iface != "" {
		query.Set("iface", in.Iface)
	}

	if in.Series != "" {
		query.Set("series", in.Series)
	}

	path := "/api/stats/history"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	return s.call(ctx, instanceInput{Instance: in.Instance}, http.MethodGet, path)
}

func (s *Server) statsNeighbors(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, in, http.MethodGet, "/api/stats/neighbors")
}

func (s *Server) statsLLDP(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, in, http.MethodGet, "/api/stats/lldp")
}

func (s *Server) statsProcesses(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, in, http.MethodGet, "/api/stats/processes")
}

func (s *Server) debugExec(ctx context.Context, _ *mcp.CallToolRequest, in execInput) (*mcp.CallToolResult, any, error) {
	i, err := s.instance(in.Instance)
	if err != nil {
		return nil, toolOutput{}, err
	}

	result, err := i.debugExec(ctx, in.Command, time.Duration(in.Timeout)*time.Second)
	if err != nil {
		return nil, toolOutput{}, err
	}

	return nil, toolOutput{Instance: in.Instance, Result: result}, nil
}

func (s *Server) wireguardKeypair(ctx context.Context, _ *mcp.CallToolRequest, in instanceInput) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, in, http.MethodPost, "/api/wireguard/keygen")
}

func (s *Server) wireguardPublicKey(ctx context.Context, _ *mcp.CallToolRequest, in privateKeyInput) (*mcp.CallToolResult, any, error) {
	i, err := s.instance(in.Instance)
	if err != nil {
		return nil, toolOutput{}, err
	}

	result, err := i.request(ctx, http.MethodPost, "/api/wireguard/pubkey", "text/plain", []byte(in.PrivateKey))
	return nil, toolOutput{Instance: in.Instance, Result: result}, err
}
