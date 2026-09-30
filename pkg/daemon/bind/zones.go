package bind

import "strings"

type ZoneRecord struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl,omitempty" validate:"optional"`
	Priority int    `json:"priority,omitempty" validate:"optional"`
}

type ZoneInput struct {
	Name    string       `json:"name"`
	Serial  int          `json:"serial,omitempty" validate:"optional"`
	Records []ZoneRecord `json:"records,omitempty" validate:"optional"`
}

type ZoneView struct {
	Name     string       `json:"name"`
	Records  []ZoneRecord `json:"records,omitempty" validate:"optional"`
	Serial   int          `json:"serial"`
	Answered bool         `json:"answered"`
}

func (c *Client) Zones(zones []ZoneInput) []ZoneView {
	out := make([]ZoneView, len(zones))
	done := make(chan int, len(zones))
	for index, zone := range zones {
		go func() {
			out[index] = c.zoneView(zone)
			done <- index
		}()
	}
	for range zones {
		<-done
	}

	return out
}

func DeclaredZones(zones []ZoneInput) []ZoneView {
	out := make([]ZoneView, 0, len(zones))
	for _, zone := range zones {
		out = append(out, ZoneView{Name: zone.Name, Records: zone.Records, Serial: zone.Serial})
	}

	return out
}

func (c *Client) zoneView(zone ZoneInput) ZoneView {
	view := ZoneView{Name: zone.Name, Records: zone.Records, Serial: zone.Serial}

	result, err := c.Query(zone.Name, "SOA")
	if err != nil {
		return view
	}

	for _, answer := range result.Answers {
		if !strings.EqualFold(answer.Type, "SOA") {
			continue
		}

		view.Answered = true
		if serial := soaSerial(answer.Data); serial > 0 {
			view.Serial = serial
		}

		break
	}

	return view
}

func soaSerial(data string) int {
	fields := strings.Fields(data)
	if len(fields) < 3 {
		return 0
	}

	return atoi(fields[2])
}
