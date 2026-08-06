package config

type Config struct {
	Version     string                `yaml:"version"                json:"version"`
	BaseDir     string                `yaml:"-"                      json:"-"`
	Hostname    string                `yaml:"hostname"               json:"hostname"`
	VRFs        map[string]*VRFConfig `yaml:"vrfs,omitempty"         json:"vrfs,omitempty" validate:"optional"`
	Interfaces  map[string]*Interface `yaml:"interfaces,omitempty"   json:"interfaces,omitempty" validate:"optional"`
	Tunnels     map[string]*Tunnel    `yaml:"tunnels,omitempty"      json:"tunnels,omitempty" validate:"optional"`
	Routing     *Routing              `yaml:"routing,omitempty"      json:"routing,omitempty" validate:"optional"`
	Wireguard   map[string]*Wireguard `yaml:"wireguard,omitempty"    json:"wireguard,omitempty" validate:"optional"`
	Nftables    *NftablesConfig       `yaml:"nftables,omitempty"     json:"nftables,omitempty" validate:"optional"`
	Sysctl      map[string]string     `yaml:"sysctl,omitempty"       json:"sysctl,omitempty" validate:"optional"`
	Users       map[string]*User      `yaml:"users,omitempty"        json:"users,omitempty" validate:"optional"`
	DNS         *DNS                  `yaml:"dns,omitempty"          json:"dns,omitempty" validate:"optional"`
	Services    map[string]*Service   `yaml:"services,omitempty"     json:"services,omitempty" validate:"optional"`
	Logging     *Logging              `yaml:"logging,omitempty"      json:"logging,omitempty" validate:"optional"`
	Friends     []*Friend             `yaml:"friends,omitempty"      json:"friends,omitempty" validate:"optional"`
	Conntrackd  *Conntrackd           `yaml:"conntrackd,omitempty"   json:"conntrackd,omitempty" validate:"optional"`
	SSH         *SSH                  `yaml:"ssh,omitempty"          json:"ssh,omitempty" validate:"optional"`
	BootModules []string              `yaml:"boot_modules,omitempty" json:"boot_modules,omitempty" validate:"optional"`
	GAI         *GAIConfig            `yaml:"gai,omitempty"          json:"gai,omitempty" validate:"optional"`
	Monitoring  *MonitoringConfig     `yaml:"monitoring,omitempty"   json:"monitoring,omitempty" validate:"optional"`
	DHCP        *DHCP                 `yaml:"dhcp,omitempty"         json:"dhcp,omitempty" validate:"optional"`
}

type DHCP struct {
	Enabled      bool             `yaml:"enabled,omitempty" json:"enabled,omitempty" validate:"optional"`
	Interfaces   []string         `yaml:"interfaces,omitempty"    json:"interfaces,omitempty" validate:"optional"`
	ControlAgent *KeaControlAgent `yaml:"control_agent,omitempty" json:"control_agent,omitempty" validate:"optional"`
	Subnets4     []KeaSubnet      `yaml:"subnets4,omitempty"      json:"subnets4,omitempty" validate:"optional"`
	Subnets6     []KeaSubnet      `yaml:"subnets6,omitempty"      json:"subnets6,omitempty" validate:"optional"`
}

type KeaControlAgent struct {
	URL      string `yaml:"url,omitempty"      json:"url,omitempty" validate:"optional"`
	User     string `yaml:"user,omitempty"     json:"user,omitempty" validate:"optional"`
	Password string `yaml:"password,omitempty" json:"password,omitempty" validate:"optional"`
}

type KeaSubnet struct {
	Subnet        string           `yaml:"subnet"                   json:"subnet"`
	Interface     string           `yaml:"interface,omitempty"      json:"interface,omitempty" validate:"optional"`
	Pools         []string         `yaml:"pools,omitempty"          json:"pools,omitempty" validate:"optional"`
	Exclusions    []string         `yaml:"exclusions,omitempty"     json:"exclusions,omitempty" validate:"optional"`
	Gateway       string           `yaml:"gateway,omitempty"        json:"gateway,omitempty" validate:"optional"`
	DNS           []string         `yaml:"dns,omitempty"            json:"dns,omitempty" validate:"optional"`
	ValidLifetime int              `yaml:"valid_lifetime,omitempty" json:"valid_lifetime,omitempty" validate:"optional"`
	Reservations  []KeaReservation `yaml:"reservations,omitempty"   json:"reservations,omitempty" validate:"optional"`
}

type KeaReservation struct {
	Hostname  string `yaml:"hostname,omitempty"   json:"hostname,omitempty" validate:"optional"`
	HWAddress string `yaml:"hw_address,omitempty" json:"hw_address,omitempty" validate:"optional"`
	DUID      string `yaml:"duid,omitempty"       json:"duid,omitempty" validate:"optional"`
	IPAddress string `yaml:"ip_address,omitempty" json:"ip_address,omitempty" validate:"optional"`
}

type MonitoringConfig struct {
	Collection CollectionIntervals `yaml:"collection,omitempty" json:"collection,omitempty" validate:"optional"`
}

type CollectionIntervals struct {
	Iface     int `yaml:"iface,omitempty"     json:"iface,omitempty" validate:"optional"`
	System    int `yaml:"system,omitempty"    json:"system,omitempty" validate:"optional"`
	BGP       int `yaml:"bgp,omitempty"       json:"bgp,omitempty" validate:"optional"`
	Proto     int `yaml:"proto,omitempty"     json:"proto,omitempty" validate:"optional"`
	Neighbors int `yaml:"neighbors,omitempty" json:"neighbors,omitempty" validate:"optional"`
	Routes    int `yaml:"routes,omitempty"    json:"routes,omitempty" validate:"optional"`
}

type Friend struct {
	Name          string         `yaml:"name"                      json:"name"`
	Hostname      string         `yaml:"hostname,omitempty"        json:"hostname,omitempty" validate:"optional"`
	URL           string         `yaml:"url"                       json:"url"`
	Token         string         `yaml:"token"                     json:"token"`
	TLSSkipVerify bool           `yaml:"tls_skip_verify,omitempty" json:"tls_skip_verify,omitempty" validate:"optional"`
	Enabled       *bool          `yaml:"enabled,omitempty"         json:"enabled,omitempty" validate:"optional"`
	Manage        bool           `yaml:"manage,omitempty"          json:"manage,omitempty" validate:"optional"`
	Identity      FriendIdentity `yaml:"identity,omitempty"        json:"identity,omitempty" validate:"optional"`
	HA            *FriendHA      `yaml:"ha,omitempty"              json:"ha,omitempty" validate:"optional"`

	Sync *FriendSync `yaml:"sync,omitempty" json:"sync,omitempty" validate:"optional"`
}

func (f *Friend) IsEnabled() bool {
	return f.Enabled == nil || *f.Enabled
}

type FriendIdentity struct {
	Fingerprint     string `yaml:"fingerprint,omitempty"      json:"fingerprint,omitempty" validate:"optional"`
	PublicKey       string `yaml:"public_key,omitempty"       json:"public_key,omitempty" validate:"optional"`
	X25519PublicKey string `yaml:"x25519_public_key,omitempty" json:"x25519_public_key,omitempty" validate:"optional"`
}

type FriendSync struct {
	Sections  []string          `yaml:"sections,omitempty"  json:"sections,omitempty" validate:"optional"`
	Overrides map[string]string `yaml:"overrides,omitempty" json:"overrides,omitempty" validate:"optional"`
}

type FriendHA struct {
	Enabled  bool          `yaml:"enabled,omitempty"  json:"enabled,omitempty" validate:"optional"`
	Link     *FriendHALink `yaml:"link,omitempty"     json:"link,omitempty" validate:"optional"`
	Priority int           `yaml:"priority,omitempty" json:"priority,omitempty" validate:"optional"`
}

type FriendHALink struct {
	Interface string `yaml:"interface"      json:"interface"`
	Address   string `yaml:"address"        json:"address"`
	Port      int    `yaml:"port,omitempty" json:"port,omitempty" validate:"optional"`
}

type Conntrackd struct {
	Interface    string   `yaml:"interface"               json:"interface"`
	Address      string   `yaml:"address"                 json:"address"`
	PeerIPs      []string `yaml:"peer_ips"                json:"peer_ips,omitempty" validate:"optional"`
	Port         int      `yaml:"port,omitempty"          json:"port,omitempty" validate:"optional"`
	AllowInbound bool     `yaml:"allow_inbound,omitempty" json:"allow_inbound,omitempty" validate:"optional"`
}

type NftablesConfig struct {
	Defines string `yaml:"defines,omitempty" json:"defines,omitempty" validate:"optional"`

	Chains map[string]*NftChain `yaml:"chains,omitempty" json:"chains,omitempty" validate:"optional"`

	Include []string `yaml:"include,omitempty" json:"include,omitempty" validate:"optional"`
}

type NftChain struct {
	Policy string `yaml:"policy,omitempty" json:"policy,omitempty" validate:"optional"`

	Rules string `yaml:"rules,omitempty" json:"rules,omitempty" validate:"optional"`

	Files []string `yaml:"files,omitempty" json:"files,omitempty" validate:"optional"`

	Managed []ManagedRule `yaml:"managed,omitempty" json:"managed,omitempty" validate:"optional"`
}

type ManagedRule struct {
	Comment  string     `yaml:"comment,omitempty"   json:"comment,omitempty" validate:"optional"`
	Tag      string     `yaml:"tag,omitempty"       json:"tag,omitempty" validate:"optional"`
	Disabled bool       `yaml:"disabled,omitempty"  json:"disabled,omitempty" validate:"optional"`
	Match    *RuleMatch `yaml:"match,omitempty"     json:"match,omitempty" validate:"optional"`
	Action   string     `yaml:"action"              json:"action"`
	ActionTo string     `yaml:"action_to,omitempty" json:"action_to,omitempty" validate:"optional"`
}

type RuleMatch struct {
	Protocol   string `yaml:"protocol,omitempty"    json:"protocol,omitempty" validate:"optional"`
	IIF        string `yaml:"iif,omitempty"         json:"iif,omitempty" validate:"optional"`
	OIF        string `yaml:"oif,omitempty"         json:"oif,omitempty" validate:"optional"`
	SAddr      string `yaml:"saddr,omitempty"       json:"saddr,omitempty" validate:"optional"`
	DAddr      string `yaml:"daddr,omitempty"       json:"daddr,omitempty" validate:"optional"`
	SPort      string `yaml:"sport,omitempty"       json:"sport,omitempty" validate:"optional"`
	DPort      string `yaml:"dport,omitempty"       json:"dport,omitempty" validate:"optional"`
	CTState    string `yaml:"ct_state,omitempty"    json:"ct_state,omitempty" validate:"optional"`
	AddrFamily string `yaml:"addr_family,omitempty" json:"addr_family,omitempty" validate:"optional"`
}

type Interface struct {
	Select      string           `yaml:"select"                json:"select"`
	Type        string           `yaml:"type,omitempty"        json:"type,omitempty" validate:"optional"`
	VRF         string           `yaml:"vrf,omitempty"         json:"vrf,omitempty" validate:"optional"`
	Addresses   []string         `yaml:"addresses,omitempty"   json:"addresses,omitempty" validate:"optional"`
	DHCPOptions []string         `yaml:"dhcp_options,omitempty" json:"dhcp_options,omitempty" validate:"optional"`
	MTU         int              `yaml:"mtu,omitempty"         json:"mtu,omitempty" validate:"optional"`
	VLANs       map[string]*VLAN `yaml:"vlans,omitempty"       json:"vlans,omitempty" validate:"optional"`
	Bridge      *Bridge          `yaml:"bridge,omitempty"      json:"bridge,omitempty" validate:"optional"`
	VRRP        []*VRRPInstance  `yaml:"vrrp,omitempty"        json:"vrrp,omitempty" validate:"optional"`
	Device      string           `yaml:"-"                     json:"-"`
}

type VRRPInstance struct {
	Name            string   `yaml:"name,omitempty"             json:"name,omitempty" validate:"optional"`
	Friend          string   `yaml:"friend,omitempty"           json:"friend,omitempty" validate:"optional"`
	ID              int      `yaml:"id"                         json:"id"`
	VIPs            []string `yaml:"vips"                       json:"vips,omitempty" validate:"optional"`
	Priority        int      `yaml:"priority,omitempty"         json:"priority,omitempty" validate:"optional"`
	Password        string   `yaml:"password,omitempty"         json:"password,omitempty" validate:"optional"`
	Interface       string   `yaml:"interface,omitempty"        json:"interface,omitempty" validate:"optional"`
	TrackInterfaces []string `yaml:"track_interfaces,omitempty" json:"track_interfaces,omitempty" validate:"optional"`
	VirtualRoutes   []string `yaml:"virtual_routes,omitempty"   json:"virtual_routes,omitempty" validate:"optional"`
	Switchover      bool     `yaml:"switchover,omitempty"       json:"switchover,omitempty" validate:"optional"`
	AllowInbound    bool     `yaml:"allow_inbound,omitempty"    json:"allow_inbound,omitempty" validate:"optional"`
}

type Bridge struct {
	Members       []string `yaml:"members,omitempty" json:"members,omitempty" validate:"optional"`
	STP           bool     `yaml:"stp,omitempty"     json:"stp,omitempty" validate:"optional"`
	MemberDevices []string `yaml:"-"                 json:"-"`
}

type VLAN struct {
	ID        int      `yaml:"id"                  json:"id"`
	Addresses []string `yaml:"addresses,omitempty" json:"addresses,omitempty" validate:"optional"`
	MTU       int      `yaml:"mtu,omitempty"       json:"mtu,omitempty" validate:"optional"`
	Device    string   `yaml:"-"                   json:"-"`
}

type Routing struct {
	Static  []StaticRoute          `yaml:"static,omitempty"  json:"static,omitempty" validate:"optional"`
	BGP     *BGP                   `yaml:"bgp,omitempty"     json:"bgp,omitempty" validate:"optional"`
	OSPF    *OSPF                  `yaml:"ospf,omitempty"    json:"ospf,omitempty" validate:"optional"`
	OSPF6   *OSPF6                 `yaml:"ospf6,omitempty"   json:"ospf6,omitempty" validate:"optional"`
	Anycast *AnycastConfig         `yaml:"anycast,omitempty" json:"anycast,omitempty" validate:"optional"`
	RADVD   *RADVDConfig           `yaml:"radvd,omitempty"   json:"radvd,omitempty" validate:"optional"`
	PBR     *PBR                   `yaml:"pbr,omitempty"     json:"pbr,omitempty" validate:"optional"`
	BFD     *BFD                   `yaml:"bfd,omitempty"     json:"bfd,omitempty" validate:"optional"`
	VRFs    map[string]*VRFRouting `yaml:"vrfs,omitempty"    json:"vrfs,omitempty" validate:"optional"`
}

type BFD struct {
	Profiles []BFDProfile `yaml:"profiles,omitempty" json:"profiles,omitempty" validate:"optional"`
}

type BFDProfile struct {
	Name             string   `yaml:"name"                          json:"name"`
	DetectMultiplier int      `yaml:"detect_multiplier,omitempty"   json:"detect_multiplier,omitempty" validate:"optional"`
	ReceiveInterval  int      `yaml:"receive_interval,omitempty"    json:"receive_interval,omitempty" validate:"optional"`
	TransmitInterval int      `yaml:"transmit_interval,omitempty"   json:"transmit_interval,omitempty" validate:"optional"`
	EchoMode         bool     `yaml:"echo_mode,omitempty"           json:"echo_mode,omitempty" validate:"optional"`
	PassiveMode      bool     `yaml:"passive_mode,omitempty"        json:"passive_mode,omitempty" validate:"optional"`
	MinimumTTL       int      `yaml:"minimum_ttl,omitempty"         json:"minimum_ttl,omitempty" validate:"optional"`
	Extra            []string `yaml:"extra,omitempty"               json:"extra,omitempty" validate:"optional"`
}

type PBR struct {
	NexthopGroups map[string]*PBRNexthopGroup `yaml:"nexthop_groups,omitempty" json:"nexthop_groups,omitempty" validate:"optional"`
	Maps          map[string][]*PBRMapEntry   `yaml:"maps,omitempty"           json:"maps,omitempty" validate:"optional"`
	Policies      map[string]string           `yaml:"policies,omitempty"       json:"policies,omitempty" validate:"optional"`
}

type PBRNexthopGroup struct {
	Nexthops []PBRNexthop `yaml:"nexthops" json:"nexthops,omitempty" validate:"optional"`
}

type PBRNexthop struct {
	Address    string `yaml:"address,omitempty"     json:"address,omitempty" validate:"optional"`
	Dev        string `yaml:"dev,omitempty"         json:"dev,omitempty" validate:"optional"`
	NexthopVRF string `yaml:"nexthop_vrf,omitempty" json:"nexthop_vrf,omitempty" validate:"optional"`
}

type PBRMapEntry struct {
	Seq             int    `yaml:"seq"                         json:"seq"`
	MatchSrc        string `yaml:"match_src,omitempty"         json:"match_src,omitempty" validate:"optional"`
	MatchDst        string `yaml:"match_dst,omitempty"         json:"match_dst,omitempty" validate:"optional"`
	SetNexthopGroup string `yaml:"set_nexthop_group,omitempty" json:"set_nexthop_group,omitempty" validate:"optional"`
	SetNexthop      string `yaml:"set_nexthop,omitempty"       json:"set_nexthop,omitempty" validate:"optional"`
}

type VRFConfig struct {
	Table int `yaml:"table" json:"table"`
}

type VRFRouting struct {
	Static []StaticRoute `yaml:"static,omitempty" json:"static,omitempty" validate:"optional"`
	BGP    *BGP          `yaml:"bgp,omitempty"    json:"bgp,omitempty" validate:"optional"`
	OSPF   *OSPF         `yaml:"ospf,omitempty"   json:"ospf,omitempty" validate:"optional"`
	OSPF6  *OSPF6        `yaml:"ospf6,omitempty"  json:"ospf6,omitempty" validate:"optional"`
}

type OSPF6 struct {
	RouterID                    string                     `yaml:"router_id,omitempty"                     json:"router_id,omitempty" validate:"optional"`
	Areas                       []OSPF6Area                `yaml:"areas,omitempty"                         json:"areas,omitempty" validate:"optional"`
	PassiveInterfaces           []string                   `yaml:"passive_interfaces,omitempty"            json:"passive_interfaces,omitempty" validate:"optional"`
	Redistribute                []string                   `yaml:"redistribute,omitempty"                  json:"redistribute,omitempty" validate:"optional"`
	Interfaces                  map[string]*OSPF6Interface `yaml:"interfaces,omitempty"                    json:"interfaces,omitempty" validate:"optional"`
	DefaultInformationOriginate bool                       `yaml:"default_information_originate,omitempty" json:"default_information_originate,omitempty" validate:"optional"`
	ReferenceBandwidth          int                        `yaml:"reference_bandwidth,omitempty"           json:"reference_bandwidth,omitempty" validate:"optional"`
	Distance                    int                        `yaml:"distance,omitempty"                      json:"distance,omitempty" validate:"optional"`
	AllowInbound                []string                   `yaml:"allow_inbound,omitempty"                 json:"allow_inbound,omitempty" validate:"optional"`
	Extra                       []string                   `yaml:"extra,omitempty"                         json:"extra,omitempty" validate:"optional"`
}

type OSPF6Area struct {
	ID            string   `yaml:"id"                        json:"id"`
	Ranges        []string `yaml:"ranges,omitempty"          json:"ranges,omitempty" validate:"optional"`
	Type          string   `yaml:"type,omitempty"            json:"type,omitempty" validate:"optional"`
	StubNoSummary bool     `yaml:"stub_no_summary,omitempty" json:"stub_no_summary,omitempty" validate:"optional"`
	DefaultCost   int      `yaml:"default_cost,omitempty"    json:"default_cost,omitempty" validate:"optional"`
	Extra         []string `yaml:"extra,omitempty"           json:"extra,omitempty" validate:"optional"`
}

type OSPF6Interface struct {
	Area               string   `yaml:"area,omitempty"                json:"area,omitempty" validate:"optional"`
	Cost               int      `yaml:"cost,omitempty"                json:"cost,omitempty" validate:"optional"`
	HelloInterval      int      `yaml:"hello_interval,omitempty"      json:"hello_interval,omitempty" validate:"optional"`
	DeadInterval       int      `yaml:"dead_interval,omitempty"       json:"dead_interval,omitempty" validate:"optional"`
	NetworkType        string   `yaml:"network_type,omitempty"        json:"network_type,omitempty" validate:"optional"`
	Priority           int      `yaml:"priority,omitempty"            json:"priority,omitempty" validate:"optional"`
	RetransmitInterval int      `yaml:"retransmit_interval,omitempty" json:"retransmit_interval,omitempty" validate:"optional"`
	TransmitDelay      int      `yaml:"transmit_delay,omitempty"      json:"transmit_delay,omitempty" validate:"optional"`
	MTUIgnore          bool     `yaml:"mtu_ignore,omitempty"          json:"mtu_ignore,omitempty" validate:"optional"`
	BFD                bool     `yaml:"bfd,omitempty"                 json:"bfd,omitempty" validate:"optional"`
	Extra              []string `yaml:"extra,omitempty"               json:"extra,omitempty" validate:"optional"`
}

type RADVDConfig struct {
	Interfaces map[string]*RADVDInterface `yaml:"interfaces,omitempty" json:"interfaces,omitempty" validate:"optional"`
}

type RADVDInterface struct {
	AdvSendAdvert        bool          `yaml:"adv_send_advert,omitempty"        json:"adv_send_advert,omitempty" validate:"optional"`
	MinRtrAdvInterval    int           `yaml:"min_rtr_adv_interval,omitempty"   json:"min_rtr_adv_interval,omitempty" validate:"optional"`
	MaxRtrAdvInterval    int           `yaml:"max_rtr_adv_interval,omitempty"   json:"max_rtr_adv_interval,omitempty" validate:"optional"`
	AdvManagedFlag       bool          `yaml:"adv_managed_flag,omitempty"       json:"adv_managed_flag,omitempty" validate:"optional"`
	AdvOtherConfigFlag   bool          `yaml:"adv_other_config_flag,omitempty"  json:"adv_other_config_flag,omitempty" validate:"optional"`
	AdvDefaultLifetime   int           `yaml:"adv_default_lifetime,omitempty"   json:"adv_default_lifetime,omitempty" validate:"optional"`
	AdvDefaultPreference string        `yaml:"adv_default_preference,omitempty" json:"adv_default_preference,omitempty" validate:"optional"`
	AdvLinkMTU           int           `yaml:"adv_link_mtu,omitempty"           json:"adv_link_mtu,omitempty" validate:"optional"`
	Prefixes             []RADVDPrefix `yaml:"prefixes,omitempty"               json:"prefixes,omitempty" validate:"optional"`
	RDNSS                *RADVDRDNSS   `yaml:"rdnss,omitempty"                  json:"rdnss,omitempty" validate:"optional"`
	Routes               []RADVDRoute  `yaml:"routes,omitempty"                 json:"routes,omitempty" validate:"optional"`
}

type RADVDPrefix struct {
	Prefix               string `yaml:"prefix"                          json:"prefix"`
	AdvOnLink            bool   `yaml:"adv_on_link,omitempty"           json:"adv_on_link,omitempty" validate:"optional"`
	AdvAutonomous        bool   `yaml:"adv_autonomous,omitempty"        json:"adv_autonomous,omitempty" validate:"optional"`
	AdvRouterAddr        bool   `yaml:"adv_router_addr,omitempty"       json:"adv_router_addr,omitempty" validate:"optional"`
	AdvValidLifetime     string `yaml:"adv_valid_lifetime,omitempty"    json:"adv_valid_lifetime,omitempty" validate:"optional"`
	AdvPreferredLifetime string `yaml:"adv_preferred_lifetime,omitempty" json:"adv_preferred_lifetime,omitempty" validate:"optional"`
}

type RADVDRDNSS struct {
	Servers  []string `yaml:"servers,omitempty"  json:"servers,omitempty" validate:"optional"`
	Lifetime int      `yaml:"lifetime,omitempty" json:"lifetime,omitempty" validate:"optional"`
}

type RADVDRoute struct {
	Prefix     string `yaml:"prefix"                json:"prefix"`
	Lifetime   int    `yaml:"lifetime,omitempty"   json:"lifetime,omitempty" validate:"optional"`
	Preference string `yaml:"preference,omitempty" json:"preference,omitempty" validate:"optional"`
}

type Tunnel struct {
	Mode      string   `yaml:"mode"                json:"mode"`
	Local     string   `yaml:"local,omitempty"     json:"local,omitempty" validate:"optional"`
	Remote    string   `yaml:"remote,omitempty"    json:"remote,omitempty" validate:"optional"`
	TTL       int      `yaml:"ttl,omitempty"       json:"ttl,omitempty" validate:"optional"`
	Addresses []string `yaml:"addresses,omitempty" json:"addresses,omitempty" validate:"optional"`
	MTU       int      `yaml:"mtu,omitempty"       json:"mtu,omitempty" validate:"optional"`
	Friend    string   `yaml:"friend,omitempty"    json:"friend,omitempty" validate:"optional"`
}

type AnycastConfig struct {
	Services []AnycastService `yaml:"services,omitempty" json:"services,omitempty" validate:"optional"`
}

type AnycastService struct {
	Name       string            `yaml:"name"                  json:"name"`
	Active     bool              `yaml:"active,omitempty"      json:"active,omitempty" validate:"optional"`
	AnycastIPs []string          `yaml:"anycast_ips,omitempty" json:"anycast_ips,omitempty" validate:"optional"`
	Endpoints  []AnycastEndpoint `yaml:"endpoints,omitempty"   json:"endpoints,omitempty" validate:"optional"`
}

type AnycastEndpoint struct {
	IP        string            `yaml:"ip"                   json:"ip"`
	Interface string            `yaml:"interface,omitempty"  json:"interface,omitempty" validate:"optional"`
	Distance  int               `yaml:"distance,omitempty"   json:"distance,omitempty" validate:"optional"`
	HTTPCheck *AnycastHTTPCheck `yaml:"http_check,omitempty" json:"http_check,omitempty" validate:"optional"`
	DNSCheck  *AnycastDNSCheck  `yaml:"dns_check,omitempty"  json:"dns_check,omitempty" validate:"optional"`
}

type AnycastHTTPCheck struct {
	Verb         string            `yaml:"verb,omitempty"          json:"verb,omitempty" validate:"optional"`
	URL          string            `yaml:"url,omitempty"           json:"url,omitempty" validate:"optional"`
	ExpectedCode int               `yaml:"expected_code,omitempty" json:"expected_code,omitempty" validate:"optional"`
	Headers      map[string]string `yaml:"headers,omitempty"       json:"headers,omitempty" validate:"optional"`
	Body         string            `yaml:"body,omitempty"          json:"body,omitempty" validate:"optional"`
	Timeout      int               `yaml:"timeout,omitempty"       json:"timeout,omitempty" validate:"optional"`
}

type AnycastDNSCheck struct {
	Resolver string `yaml:"resolver,omitempty" json:"resolver,omitempty" validate:"optional"`
	Type     string `yaml:"type,omitempty"     json:"type,omitempty" validate:"optional"`
	Query    string `yaml:"query,omitempty"    json:"query,omitempty" validate:"optional"`
	Expected string `yaml:"expected,omitempty" json:"expected,omitempty" validate:"optional"`
	Timeout  int    `yaml:"timeout,omitempty"  json:"timeout,omitempty" validate:"optional"`
}

type StaticRoute struct {
	Destination string `yaml:"destination"        json:"destination"`
	Via         string `yaml:"via,omitempty"      json:"via,omitempty" validate:"optional"`
	Dev         string `yaml:"dev,omitempty"      json:"dev,omitempty" validate:"optional"`
	Metric      int    `yaml:"metric,omitempty"   json:"metric,omitempty" validate:"optional"`
}

type BGP struct {
	ASN                  int                          `yaml:"asn"                             json:"asn"`
	RouterID             string                       `yaml:"router_id,omitempty"             json:"router_id,omitempty" validate:"optional"`
	NoEBGPRequiresPolicy bool                         `yaml:"no_ebgp_requires_policy,omitempty" json:"no_ebgp_requires_policy,omitempty" validate:"optional"`
	NoDefaultIPv4Unicast bool                         `yaml:"no_default_ipv4_unicast,omitempty" json:"no_default_ipv4_unicast,omitempty" validate:"optional"`
	NoImportCheck        bool                         `yaml:"no_import_check,omitempty"        json:"no_import_check,omitempty" validate:"optional"`
	AllowInbound         []string                     `yaml:"allow_inbound,omitempty"         json:"allow_inbound,omitempty" validate:"optional"`
	Neighbors            []BGPNeighbor                `yaml:"neighbors,omitempty"             json:"neighbors,omitempty" validate:"optional"`
	AddressFamilies      map[string]*BGPAddressFamily `yaml:"address_families,omitempty"      json:"address_families,omitempty" validate:"optional"`
	PrefixLists          map[string][]PrefixEntry     `yaml:"prefix_lists,omitempty"          json:"prefix_lists,omitempty" validate:"optional"`
	RouteMaps            map[string][]RouteMapEntry   `yaml:"route_maps,omitempty"            json:"route_maps,omitempty" validate:"optional"`
	Extra                []string                     `yaml:"extra,omitempty"                 json:"extra,omitempty" validate:"optional"`
}

type BGPAddressFamily struct {
	Networks         []string `yaml:"networks,omitempty"          json:"networks,omitempty" validate:"optional"`
	Redistribute     []string `yaml:"redistribute,omitempty"      json:"redistribute,omitempty" validate:"optional"`
	ImportVRF        []string `yaml:"import_vrf,omitempty"        json:"import_vrf,omitempty" validate:"optional"`
	RouteMapIn       string   `yaml:"route_map_in,omitempty"      json:"route_map_in,omitempty" validate:"optional"`
	RouteMapOut      string   `yaml:"route_map_out,omitempty"     json:"route_map_out,omitempty" validate:"optional"`
	DefaultOriginate bool     `yaml:"default_originate,omitempty" json:"default_originate,omitempty" validate:"optional"`
	MaximumPaths     int      `yaml:"maximum_paths,omitempty"     json:"maximum_paths,omitempty" validate:"optional"`
	Extra            []string `yaml:"extra,omitempty"             json:"extra,omitempty" validate:"optional"`
}

type BGPNeighbor struct {
	Address               string                    `yaml:"address"                          json:"address"`
	RemoteASN             int                       `yaml:"remote_asn"                       json:"remote_asn"`
	EBGPMultihop          int                       `yaml:"ebgp_multihop,omitempty" json:"ebgp_multihop,omitempty" validate:"optional"`
	Description           string                    `yaml:"description,omitempty"            json:"description,omitempty" validate:"optional"`
	Password              string                    `yaml:"password,omitempty"               json:"password,omitempty" validate:"optional"`
	UpdateSource          string                    `yaml:"update_source,omitempty"          json:"update_source,omitempty" validate:"optional"`
	DisableConnectedCheck bool                      `yaml:"disable_connected_check,omitempty" json:"disable_connected_check,omitempty" validate:"optional"`
	Passive               bool                      `yaml:"passive,omitempty"                json:"passive,omitempty" validate:"optional"`
	Shutdown              bool                      `yaml:"shutdown,omitempty"               json:"shutdown,omitempty" validate:"optional"`
	BFD                   bool                      `yaml:"bfd,omitempty"                    json:"bfd,omitempty" validate:"optional"`
	BFDProfile            string                    `yaml:"bfd_profile,omitempty"            json:"bfd_profile,omitempty" validate:"optional"`
	Extra                 []string                  `yaml:"extra,omitempty"                  json:"extra,omitempty" validate:"optional"`
	AddressFamilies       map[string]*BGPNeighborAF `yaml:"address_families,omitempty"       json:"address_families,omitempty" validate:"optional"`
}

type BGPNeighborAF struct {
	Disabled             bool     `yaml:"disabled,omitempty"                   json:"disabled,omitempty" validate:"optional"`
	SoftReconfiguration  bool     `yaml:"soft_reconfiguration,omitempty"       json:"soft_reconfiguration,omitempty" validate:"optional"`
	NextHopSelf          bool     `yaml:"next_hop_self,omitempty"              json:"next_hop_self,omitempty" validate:"optional"`
	RouteReflectorClient bool     `yaml:"route_reflector_client,omitempty"     json:"route_reflector_client,omitempty" validate:"optional"`
	RemovePrivateAS      bool     `yaml:"remove_private_as,omitempty"          json:"remove_private_as,omitempty" validate:"optional"`
	AllowASIn            int      `yaml:"allowas_in,omitempty"                 json:"allowas_in,omitempty" validate:"optional"`
	Weight               int      `yaml:"weight,omitempty"                     json:"weight,omitempty" validate:"optional"`
	PrefixListIn         string   `yaml:"prefix_list_in,omitempty"             json:"prefix_list_in,omitempty" validate:"optional"`
	PrefixListOut        string   `yaml:"prefix_list_out,omitempty"            json:"prefix_list_out,omitempty" validate:"optional"`
	RouteMapIn           string   `yaml:"route_map_in,omitempty"               json:"route_map_in,omitempty" validate:"optional"`
	RouteMapOut          string   `yaml:"route_map_out,omitempty"              json:"route_map_out,omitempty" validate:"optional"`
	DefaultOriginate     bool     `yaml:"default_originate,omitempty"          json:"default_originate,omitempty" validate:"optional"`
	Extra                []string `yaml:"extra,omitempty"                      json:"extra,omitempty" validate:"optional"`
}

type PrefixEntry struct {
	Seq    int    `yaml:"seq"           json:"seq"`
	Action string `yaml:"action"        json:"action"`
	Prefix string `yaml:"prefix"        json:"prefix"`
	GE     int    `yaml:"ge,omitempty"  json:"ge,omitempty" validate:"optional"`
	LE     int    `yaml:"le,omitempty"  json:"le,omitempty" validate:"optional"`
}

type RouteMapEntry struct {
	Seq      int               `yaml:"seq"               json:"seq"`
	Action   string            `yaml:"action"            json:"action"`
	Match    map[string]string `yaml:"match,omitempty"   json:"match,omitempty" validate:"optional"`
	Set      map[string]string `yaml:"set,omitempty"     json:"set,omitempty" validate:"optional"`
	Call     string            `yaml:"call,omitempty"     json:"call,omitempty" validate:"optional"`
	OnMatch  string            `yaml:"on_match,omitempty" json:"on_match,omitempty" validate:"optional"`
	Continue int               `yaml:"continue,omitempty" json:"continue,omitempty" validate:"optional"`
}

type OSPF struct {
	RouterID                    string                    `yaml:"router_id,omitempty"                      json:"router_id,omitempty" validate:"optional"`
	Areas                       []OSPFArea                `yaml:"areas,omitempty"                          json:"areas,omitempty" validate:"optional"`
	PassiveInterfaces           []string                  `yaml:"passive_interfaces,omitempty"             json:"passive_interfaces,omitempty" validate:"optional"`
	Redistribute                []string                  `yaml:"redistribute,omitempty"                   json:"redistribute,omitempty" validate:"optional"`
	Interfaces                  map[string]*OSPFInterface `yaml:"interfaces,omitempty"                     json:"interfaces,omitempty" validate:"optional"`
	DefaultInformationOriginate bool                      `yaml:"default_information_originate,omitempty"  json:"default_information_originate,omitempty" validate:"optional"`
	ReferenceBandwidth          int                       `yaml:"reference_bandwidth,omitempty"            json:"reference_bandwidth,omitempty" validate:"optional"`
	Distance                    int                       `yaml:"distance,omitempty"                       json:"distance,omitempty" validate:"optional"`
	AllowInbound                []string                  `yaml:"allow_inbound,omitempty"                  json:"allow_inbound,omitempty" validate:"optional"`
	Extra                       []string                  `yaml:"extra,omitempty"                          json:"extra,omitempty" validate:"optional"`
}

type OSPFArea struct {
	ID            string   `yaml:"id"                       json:"id"`
	Networks      []string `yaml:"networks,omitempty"       json:"networks,omitempty" validate:"optional"`
	Type          string   `yaml:"type,omitempty"           json:"type,omitempty" validate:"optional"`
	StubNoSummary bool     `yaml:"stub_no_summary,omitempty" json:"stub_no_summary,omitempty" validate:"optional"`
	Ranges        []string `yaml:"ranges,omitempty"         json:"ranges,omitempty" validate:"optional"`
	DefaultCost   int      `yaml:"default_cost,omitempty"   json:"default_cost,omitempty" validate:"optional"`
	Auth          string   `yaml:"auth,omitempty"           json:"auth,omitempty" validate:"optional"`
	Extra         []string `yaml:"extra,omitempty"          json:"extra,omitempty" validate:"optional"`
}

type OSPFInterface struct {
	Area               string   `yaml:"area,omitempty"                json:"area,omitempty" validate:"optional"`
	Cost               int      `yaml:"cost,omitempty"                json:"cost,omitempty" validate:"optional"`
	HelloInterval      int      `yaml:"hello_interval,omitempty"      json:"hello_interval,omitempty" validate:"optional"`
	DeadInterval       int      `yaml:"dead_interval,omitempty"       json:"dead_interval,omitempty" validate:"optional"`
	NetworkType        string   `yaml:"network_type,omitempty"        json:"network_type,omitempty" validate:"optional"`
	Priority           int      `yaml:"priority,omitempty"            json:"priority,omitempty" validate:"optional"`
	RetransmitInterval int      `yaml:"retransmit_interval,omitempty" json:"retransmit_interval,omitempty" validate:"optional"`
	TransmitDelay      int      `yaml:"transmit_delay,omitempty"      json:"transmit_delay,omitempty" validate:"optional"`
	MTUIgnore          bool     `yaml:"mtu_ignore,omitempty"          json:"mtu_ignore,omitempty" validate:"optional"`
	BFD                bool     `yaml:"bfd,omitempty"                 json:"bfd,omitempty" validate:"optional"`
	AuthType           string   `yaml:"auth_type,omitempty"           json:"auth_type,omitempty" validate:"optional"`
	AuthKey            string   `yaml:"auth_key,omitempty"            json:"auth_key,omitempty" validate:"optional"`
	AuthKeyID          int      `yaml:"auth_key_id,omitempty"         json:"auth_key_id,omitempty" validate:"optional"`
	Extra              []string `yaml:"extra,omitempty"               json:"extra,omitempty" validate:"optional"`
}

type Wireguard struct {
	Friend         string   `yaml:"friend,omitempty"           json:"friend,omitempty" validate:"optional"`
	ListenPort     int      `yaml:"listen_port,omitempty"      json:"listen_port,omitempty" validate:"optional"`
	PrivateKey     string   `yaml:"private_key,omitempty"      json:"private_key,omitempty" validate:"optional"`
	PrivateKeyFile string   `yaml:"private_key_file,omitempty" json:"private_key_file,omitempty" validate:"optional"`
	Table          string   `yaml:"table,omitempty"            json:"table,omitempty" validate:"optional"`
	Addresses      []string `yaml:"addresses,omitempty"        json:"addresses,omitempty" validate:"optional"`
	MTU            int      `yaml:"mtu,omitempty"              json:"mtu,omitempty" validate:"optional"`
	PreUp          []string `yaml:"pre_up,omitempty"           json:"pre_up,omitempty" validate:"optional"`
	PostUp         []string `yaml:"post_up,omitempty"          json:"post_up,omitempty" validate:"optional"`
	PreDown        []string `yaml:"pre_down,omitempty"         json:"pre_down,omitempty" validate:"optional"`
	PostDown       []string `yaml:"post_down,omitempty"        json:"post_down,omitempty" validate:"optional"`
	Peers          []WGPeer `yaml:"peers,omitempty"            json:"peers,omitempty" validate:"optional"`

	AllowInbound bool `yaml:"allow_inbound,omitempty" json:"allow_inbound,omitempty" validate:"optional"`
}

type WGPeer struct {
	Name             string   `yaml:"name,omitempty"               json:"name,omitempty" validate:"optional"`
	PrivateKey       string   `yaml:"private_key,omitempty"        json:"private_key,omitempty" validate:"optional"`
	PublicKey        string   `yaml:"public_key,omitempty"         json:"public_key,omitempty" validate:"optional"`
	PresharedKey     string   `yaml:"preshared_key,omitempty"      json:"preshared_key,omitempty" validate:"optional"`
	PresharedKeyFile string   `yaml:"preshared_key_file,omitempty" json:"preshared_key_file,omitempty" validate:"optional"`
	Endpoint         string   `yaml:"endpoint,omitempty"           json:"endpoint,omitempty" validate:"optional"`
	AllowedIPs       []string `yaml:"allowed_ips,omitempty"        json:"allowed_ips,omitempty" validate:"optional"`
	Keepalive        int      `yaml:"keepalive,omitempty"          json:"keepalive,omitempty" validate:"optional"`
}

type Logging struct {
	Target string `yaml:"target,omitempty" json:"target,omitempty" validate:"optional"`
	File   string `yaml:"file,omitempty"   json:"file,omitempty" validate:"optional"`
	Level  string `yaml:"level,omitempty"  json:"level,omitempty" validate:"optional"`
}

type User struct {
	UID          int      `yaml:"uid,omitempty"      json:"uid,omitempty" validate:"optional"`
	Shell        string   `yaml:"shell,omitempty"    json:"shell,omitempty" validate:"optional"`
	Home         string   `yaml:"home,omitempty"     json:"home,omitempty" validate:"optional"`
	Groups       []string `yaml:"groups,omitempty"   json:"groups,omitempty" validate:"optional"`
	SSHKeys      []string `yaml:"ssh_keys,omitempty" json:"ssh_keys,omitempty" validate:"optional"`
	System       bool     `yaml:"system,omitempty"   json:"system,omitempty" validate:"optional"`
	PasswordHash string   `yaml:"password_hash,omitempty" json:"password_hash,omitempty" validate:"optional"`
}

type DNS struct {
	Nameservers []string `yaml:"nameservers,omitempty" json:"nameservers,omitempty" validate:"optional"`
	Search      []string `yaml:"search,omitempty"      json:"search,omitempty" validate:"optional"`
}

type SSH struct {
	Port                int      `yaml:"port,omitempty"                  json:"port,omitempty" validate:"optional"`
	PermitRootLogin     string   `yaml:"permit_root_login,omitempty"     json:"permit_root_login,omitempty" validate:"optional"`
	PasswordAuth        string   `yaml:"password_auth,omitempty"         json:"password_auth,omitempty" validate:"optional"`
	PubkeyAuth          string   `yaml:"pubkey_auth,omitempty"           json:"pubkey_auth,omitempty" validate:"optional"`
	AllowTcpForwarding  string   `yaml:"allow_tcp_forwarding,omitempty"  json:"allow_tcp_forwarding,omitempty" validate:"optional"`
	X11Forwarding       string   `yaml:"x11_forwarding,omitempty"        json:"x11_forwarding,omitempty" validate:"optional"`
	MaxAuthTries        int      `yaml:"max_auth_tries,omitempty"        json:"max_auth_tries,omitempty" validate:"optional"`
	LoginGraceTime      int      `yaml:"login_grace_time,omitempty"      json:"login_grace_time,omitempty" validate:"optional"`
	ClientAliveInterval int      `yaml:"client_alive_interval,omitempty" json:"client_alive_interval,omitempty" validate:"optional"`
	ClientAliveCountMax int      `yaml:"client_alive_count_max,omitempty" json:"client_alive_count_max,omitempty" validate:"optional"`
	AllowUsers          []string `yaml:"allow_users,omitempty"           json:"allow_users,omitempty" validate:"optional"`
	AllowGroups         []string `yaml:"allow_groups,omitempty"          json:"allow_groups,omitempty" validate:"optional"`
	Banner              string   `yaml:"banner,omitempty"                json:"banner,omitempty" validate:"optional"`
}

type Service struct {
	Enable  bool            `yaml:"enable"              json:"enable"`
	Configs []ServiceConfig `yaml:"configs,omitempty"   json:"configs,omitempty" validate:"optional"`
	After   string          `yaml:"after,omitempty"     json:"after,omitempty" validate:"optional"`
}

type ServiceConfig struct {
	Src  string `yaml:"src"            json:"src"`
	Dest string `yaml:"dest"           json:"dest"`
	Mode int    `yaml:"mode,omitempty" json:"mode,omitempty" validate:"optional"`
}

type GAIConfig struct {
	Precedence []GAIEntry `yaml:"precedence,omitempty" json:"precedence,omitempty" validate:"optional"`
	Label      []GAIEntry `yaml:"label,omitempty"      json:"label,omitempty" validate:"optional"`
}

type GAIEntry struct {
	Prefix string `yaml:"prefix" json:"prefix"`
	Value  int    `yaml:"value"  json:"value"`
}
