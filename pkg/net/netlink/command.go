//go:build linux

package netlink

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	vnl "github.com/vishvananda/netlink"
)

func RunCommand(operation, target string, command func() error) error {
	start := time.Now()
	log.Info().Str("operation", operation).Str("target", target).Msg("netlink execute")
	err := command()
	event := log.Info()
	if err != nil {
		event = log.Error().Err(err)
	}

	event.Str("operation", operation).
		Str("target", target).
		Dur("duration", time.Since(start)).
		Bool("success", err == nil).
		Msg("netlink result")
	return err
}

func linkAdd(link vnl.Link) error {
	return RunCommand("LinkAdd", link.Attrs().Name, func() error {
		return vnl.LinkAdd(link)
	})
}

func linkDel(link vnl.Link) error {
	return RunCommand("LinkDel", link.Attrs().Name, func() error {
		return vnl.LinkDel(link)
	})
}

func linkModify(link vnl.Link) error {
	return RunCommand("LinkModify", link.Attrs().Name, func() error {
		return vnl.LinkModify(link)
	})
}

func linkSetDown(link vnl.Link) error {
	return RunCommand("LinkSetDown", link.Attrs().Name, func() error {
		return vnl.LinkSetDown(link)
	})
}

func linkSetUp(link vnl.Link) error {
	return RunCommand("LinkSetUp", link.Attrs().Name, func() error {
		return vnl.LinkSetUp(link)
	})
}

func linkSetNoMaster(link vnl.Link) error {
	return RunCommand("LinkSetNoMaster", link.Attrs().Name, func() error {
		return vnl.LinkSetNoMaster(link)
	})
}

func linkSetAlias(link vnl.Link, alias string) error {
	return RunCommand("LinkSetAlias", link.Attrs().Name+" alias="+alias, func() error {
		return vnl.LinkSetAlias(link, alias)
	})
}

func linkSetMTU(link vnl.Link, mtu int) error {
	return RunCommand("LinkSetMTU", fmt.Sprintf("%s mtu=%d", link.Attrs().Name, mtu), func() error {
		return vnl.LinkSetMTU(link, mtu)
	})
}

func linkSetMaster(link, master vnl.Link) error {
	return RunCommand("LinkSetMaster", link.Attrs().Name+" master="+master.Attrs().Name, func() error {
		return vnl.LinkSetMaster(link, master)
	})
}

func addrAdd(link vnl.Link, addr *vnl.Addr) error {
	return RunCommand("AddrAdd", link.Attrs().Name+" "+addr.String(), func() error {
		return vnl.AddrAdd(link, addr)
	})
}

func addrDel(link vnl.Link, addr *vnl.Addr) error {
	return RunCommand("AddrDel", link.Attrs().Name+" "+addr.String(), func() error {
		return vnl.AddrDel(link, addr)
	})
}

func routeAdd(route *vnl.Route) error {
	return RunCommand("RouteAdd", route.String(), func() error {
		return vnl.RouteAdd(route)
	})
}

func routeDel(route *vnl.Route) error {
	return RunCommand("RouteDel", route.String(), func() error {
		return vnl.RouteDel(route)
	})
}

func routeReplace(route *vnl.Route) error {
	return RunCommand("RouteReplace", route.String(), func() error {
		return vnl.RouteReplace(route)
	})
}
