// Package escl discovers eSCL network scanners advertised through mDNS.
package escl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

const discoveryTimeout = 2 * time.Second

type service struct {
	name   string
	scheme string
}

var services = []service{
	{name: "_uscan._tcp", scheme: "http"},
	{name: "_uscans._tcp", scheme: "https"},
}

// Device is an eSCL scanner advertised on the local network.
type Device struct {
	ID      string
	Name    string
	BaseURL *url.URL
}

type discoveryResult struct {
	devices []Device
	err     error
}

type discoveryInterface struct {
	interfaceRef *net.Interface
	ipv4         bool
	ipv6         bool
}

// Discover returns eSCL scanners advertised through local mDNS services.
func Discover(ctx context.Context) ([]Device, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	queryCtx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()

	interfaces, err := discoveryInterfaces()
	if err != nil {
		return nil, fmt.Errorf("list multicast interfaces: %w", err)
	}

	results := make(chan discoveryResult, len(services)*len(interfaces))
	for _, service := range services {
		for _, network := range interfaces {
			go func() {
				devices, err := discoverService(queryCtx, service, network)
				results <- discoveryResult{devices: devices, err: err}
			}()
		}
	}

	devicesByID := map[string]Device{}
	var queryErrors []error
	for range len(services) * len(interfaces) {
		result := <-results
		if result.err != nil {
			queryErrors = append(queryErrors, result.err)
		}
		for _, device := range result.devices {
			devicesByID[device.ID] = device
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	devices := make([]Device, 0, len(devicesByID))
	for _, device := range devicesByID {
		devices = append(devices, device)
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	if len(devices) == 0 && len(queryErrors) != 0 {
		return nil, fmt.Errorf("query eSCL services: %w", errors.Join(queryErrors...))
	}

	return devices, nil
}

func discoveryInterfaces() ([]discoveryInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var networks []discoveryInterface
	for index := range interfaces {
		iface := &interfaces[index]
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		network := discoveryInterface{interfaceRef: iface}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() || ipNet.IP.IsUnspecified() {
				continue
			}
			if ipNet.IP.To4() != nil {
				network.ipv4 = true
			} else if ipNet.IP.To16() != nil {
				network.ipv6 = true
			}
		}
		if network.ipv4 || network.ipv6 {
			networks = append(networks, network)
		}
	}

	if len(networks) == 0 {
		return []discoveryInterface{{ipv4: true, ipv6: true}}, nil
	}
	return networks, nil
}

func discoverService(ctx context.Context, service service, network discoveryInterface) ([]Device, error) {
	entries := make(chan *mdns.ServiceEntry, 64)
	params := &mdns.QueryParam{
		Service:     service.name,
		Domain:      "local",
		Timeout:     discoveryTimeout,
		Entries:     entries,
		Interface:   network.interfaceRef,
		DisableIPv4: !network.ipv4,
		DisableIPv6: !network.ipv6,
		Logger:      log.New(io.Discard, "", 0),
	}

	done := make(chan error, 1)
	go func() {
		done <- mdns.QueryContext(ctx, params)
	}()

	var devices []Device
	for {
		select {
		case entry := <-entries:
			if entry == nil {
				continue
			}
			if device, ok := deviceFromEntry(entry, service.scheme); ok {
				devices = append(devices, device)
			}
		case err := <-done:
			for {
				select {
				case entry := <-entries:
					if entry != nil {
						if device, ok := deviceFromEntry(entry, service.scheme); ok {
							devices = append(devices, device)
						}
					}
				default:
					return devices, err
				}
			}
		case <-ctx.Done():
			return devices, nil
		}
	}
}

func deviceFromEntry(entry *mdns.ServiceEntry, scheme string) (Device, bool) {
	host := strings.TrimSuffix(strings.TrimSpace(entry.Host), ".")
	if host == "" || entry.Port <= 0 {
		return Device{}, false
	}

	name := strings.TrimSuffix(strings.TrimSpace(entry.Name), ".")
	resource := "/eSCL"
	for _, field := range entry.InfoFields {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(key)) {
		case "rs":
			resource = strings.TrimSpace(value)
		case "ty":
			if value = strings.TrimSpace(value); value != "" {
				name = value
			}
		}
	}

	resourceURL, err := url.Parse(resource)
	if err != nil || resourceURL.IsAbs() || resourceURL.Host != "" {
		return Device{}, false
	}
	resourceURL.Path = "/" + strings.Trim(strings.TrimSpace(resourceURL.Path), "/")
	if resourceURL.Path == "/" {
		resourceURL.Path = "/eSCL"
	}

	urlHost := host
	if (scheme == "http" && entry.Port != 80) || (scheme == "https" && entry.Port != 443) {
		urlHost = net.JoinHostPort(host, strconv.Itoa(entry.Port))
	} else if net.ParseIP(host) != nil && strings.Contains(host, ":") {
		urlHost = "[" + host + "]"
	}

	baseURL := &url.URL{
		Scheme:   scheme,
		Host:     urlHost,
		Path:     resourceURL.Path,
		RawQuery: resourceURL.RawQuery,
	}
	return Device{
		ID:      baseURL.String(),
		Name:    name,
		BaseURL: baseURL,
	}, true
}
