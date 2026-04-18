package resolver

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"strings"

	"github.com/metacubex/mihomo/common/utils"
	"github.com/metacubex/mihomo/component/trie"
	"github.com/zhangyunhao116/fastrand"
)

type Hosts struct {
	*trie.DomainTrie[HostValue]
}

func NewHosts(hosts *trie.DomainTrie[HostValue]) Hosts {
	return Hosts{
		hosts,
	}
}

// lookupStaticHost looks up the addresses and canonical host name from /etc/hosts.
func lookupStaticHost(host string) ([]string, string) {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	buf, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return nil, ""
	}

	addrs := make([]string, 0)
	canonical := ""
	for _, line := range strings.Split(string(buf), "\n") {
		if idx := strings.IndexByte(line, '#'); idx >= 0 {
			line = line[:idx]
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ip := net.ParseIP(fields[0])
		if ip == nil {
			continue
		}
		entryCanonical := strings.TrimSuffix(strings.ToLower(fields[1]), ".")
		for _, alias := range fields[1:] {
			if strings.TrimSuffix(strings.ToLower(alias), ".") != host {
				continue
			}
			addrs = append(addrs, ip.String())
			if canonical == "" {
				canonical = entryCanonical
			}
			break
		}
	}

	return addrs, canonical
}

// Return the search result and whether to match the parameter `isDomain`
func (h *Hosts) Search(domain string, isDomain bool) (*HostValue, bool) {
	if value := h.DomainTrie.Search(domain); value != nil {
		hostValue := value.Data()
		for {
			if isDomain && hostValue.IsDomain {
				return &hostValue, true
			} else {
				if node := h.DomainTrie.Search(hostValue.Domain); node != nil {
					hostValue = node.Data()
				} else {
					break
				}
			}
		}
		if isDomain == hostValue.IsDomain {
			return &hostValue, true
		}

		return &hostValue, false
	}
	if !isDomain {
		addr, _ := lookupStaticHost(domain)
		if hostValue, err := NewHostValue(addr); err == nil {
			return &hostValue, true
		}
	}
	return nil, false
}

type HostValue struct {
	IsDomain bool
	IPs      []netip.Addr
	Domain   string
}

func NewHostValue(value any) (HostValue, error) {
	isDomain := true
	ips := make([]netip.Addr, 0)
	domain := ""
	if valueArr, err := utils.ToStringSlice(value); err != nil {
		return HostValue{}, err
	} else {
		if len(valueArr) > 1 {
			isDomain = false
			for _, str := range valueArr {
				if ip, err := netip.ParseAddr(str); err == nil {
					ips = append(ips, ip)
				} else {
					return HostValue{}, err
				}
			}
		} else if len(valueArr) == 1 {
			host := valueArr[0]
			if ip, err := netip.ParseAddr(host); err == nil {
				ips = append(ips, ip)
				isDomain = false
			} else {
				domain = host
			}
		}
	}
	if isDomain {
		return NewHostValueByDomain(domain)
	} else {
		return NewHostValueByIPs(ips)
	}
}

func NewHostValueByIPs(ips []netip.Addr) (HostValue, error) {
	if len(ips) == 0 {
		return HostValue{}, errors.New("ip list is empty")
	}
	return HostValue{
		IsDomain: false,
		IPs:      ips,
	}, nil
}

func NewHostValueByDomain(domain string) (HostValue, error) {
	domain = strings.Trim(domain, ".")
	item := strings.Split(domain, ".")
	if len(item) < 2 {
		return HostValue{}, errors.New("invaild domain")
	}
	return HostValue{
		IsDomain: true,
		Domain:   domain,
	}, nil
}

func (hv HostValue) RandIP() (netip.Addr, error) {
	if hv.IsDomain {
		return netip.Addr{}, errors.New("value type is error")
	}
	return hv.IPs[fastrand.Intn(len(hv.IPs))], nil
}
