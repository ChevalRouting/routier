package types

type Flow struct {
	ID      int64  `json:"id"`
	Start   int64  `json:"start"`
	End     int64  `json:"end"`
	InIf    string `json:"in_if"`
	OutIf   string `json:"out_if"`
	Family  int    `json:"family"`
	Proto   int    `json:"proto"`
	Src     string `json:"src"`
	Dst     string `json:"dst"`
	SrcPort int    `json:"src_port"`
	DstPort int    `json:"dst_port"`
	Packets int64  `json:"packets"`
	Bytes   int64  `json:"bytes"`
	Samples int64  `json:"samples"`
}

type FlowTopEntry struct {
	Key     string `json:"key"`
	Bytes   int64  `json:"bytes"`
	Packets int64  `json:"packets"`
	Flows   int64  `json:"flows"`
}

type FlowSeriesPoint struct {
	Ts      int64  `json:"ts"`
	Iface   string `json:"iface"`
	Bytes   int64  `json:"bytes"`
	Packets int64  `json:"packets"`
}

type FlowCollectorStatus struct {
	Address string `json:"address"`
	Source  string `json:"source,omitempty" validate:"optional"`
	Sent    uint64 `json:"sent"`
	Errors  uint64 `json:"errors"`
	Error   string `json:"error,omitempty" validate:"optional"`
}

type FlowStatus struct {
	Enabled    bool                  `json:"enabled"`
	Running    bool                  `json:"running"`
	Sampling   int                   `json:"sampling"`
	Interfaces []string              `json:"interfaces"`
	Collectors []FlowCollectorStatus `json:"collectors"`
	Samples    uint64                `json:"samples"`
	Overruns   uint64                `json:"overruns"`
	Stored     int64                 `json:"stored"`
	Error      string                `json:"error,omitempty" validate:"optional"`
}

type FlowsResponse struct {
	Flows []Flow `json:"flows"`
}

type FlowTopResponse struct {
	Dimension string         `json:"dimension"`
	Entries   []FlowTopEntry `json:"entries"`
}

type FlowSeriesResponse struct {
	Bucket int64             `json:"bucket"`
	Points []FlowSeriesPoint `json:"points"`
}

type FlowPath struct {
	Keys    []string `json:"keys"`
	Bytes   int64    `json:"bytes"`
	Packets int64    `json:"packets"`
}

type FlowSankeyResponse struct {
	Dimensions []string          `json:"dimensions"`
	Total      int64             `json:"total"`
	Paths      []FlowPath        `json:"paths"`
	Names      map[string]string `json:"names"`
}
