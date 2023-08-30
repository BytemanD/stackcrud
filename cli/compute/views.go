package compute

import (
	"fmt"
	"os"
	"strings"

	"github.com/BytemanD/easygo/pkg/global/logging"
	"github.com/BytemanD/stackcrud/common"
	"github.com/BytemanD/stackcrud/openstack/compute"
	"github.com/jedib0t/go-pretty/v6/list"
)

func printServer(server compute.Server) {
	pt := common.PrettyItemTable{
		Item: server,
		ShortFields: []common.Column{
			{Name: "Id"}, {Name: "Name"}, {Name: "Description"},
			{Name: "Flavor:original_name", Text: "Flavor:original_name",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Server)
					return p.Flavor.OriginalName
				}},
			{Name: "Flavor:ram", Text: "Flavor:ram",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Server)
					return p.Flavor.Ram
				}},
			{Name: "Flavor:vcpus", Text: "Flavor:vcpus",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Server)
					return p.Flavor.Vcpus
				}},
			{Name: "Flavor:disk", Text: "Flavor:disk",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Server)
					return p.Flavor.Disk
				}},
			{Name: "Flavor:swap", Text: "Flavor:swap",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Server)
					return p.Flavor.Swap
				}},
			{Name: "Flavor:extra_specs", Text: "Flavor:extra_specs",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Server)
					return p.GetFlavorExtraSpecsString()

				}},
			{Name: "Image",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Server)
					return p.Image.Id
				}},
			{Name: "AZ", Text: "AZ"}, {Name: "Host"},
			{Name: "Status"}, {Name: "TaskState"}, {Name: "PowerState"},
			{Name: "RootBdmType"},
			{Name: "Created"}, {Name: "Updated"},
			{Name: "UserId"}, {Name: "LaunchedAt"},

			{Name: "Fault:code",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Server)
					return p.Fault.Code
				}},
			{Name: "Fault:message",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Server)
					return p.Fault.Message
				}},
			{Name: "Fault:details",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Server)
					return p.Fault.Details
				}},
		},
	}
	common.PrintPrettyItemTable(pt)
}
func printFlavor(server compute.Flavor) {
	pt := common.PrettyItemTable{
		Item: server,
		ShortFields: []common.Column{
			{Name: "Id"}, {Name: "Name"},
			{Name: "Vcpus"}, {Name: "Ram"}, {Name: "Disk"}, {Name: "Swap"},
			{Name: "RXTXFactor", Text: "RXTXFactor"},
			{Name: "OS-FLV-EXT-DATA:ephemeral", Text: "OS-FLV-EXT-DATA:ephemeral",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Flavor)
					return p.Ephemeral
				}},
			{Name: "os-flavor-access:is_public", Text: "os-flavor-access:is_public",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Flavor)
					return p.IsPublic
				}},
			{Name: "OS-FLV-DISABLED:disabled", Text: "OS-FLV-DISABLED:disabled",
				Slot: func(item interface{}) interface{} {
					p, _ := item.(compute.Flavor)
					return p.Disabled
				}},
			{Name: "ExtraSpecs", Slot: func(item interface{}) interface{} {
				p, _ := item.(compute.Flavor)
				return strings.Join(p.ExtraSpecs.GetList(), "\n")
			}},
		},
	}
	common.PrintPrettyItemTable(pt)
}

type AZHost struct {
	ZoneName         string
	ZoneState        string
	HostName         string
	ServiceName      string
	ServiceAvailable string
	ServiceStatus    string
	ServiceUpdatedAt string
}

func printAZInfo(azList []compute.AvailabilityZone) {
	azHostList := []AZHost{}
	for _, az := range azList {
		for hostName, services := range az.Hosts {
			for serviceName, service := range services {
				azHost := AZHost{
					ZoneName:         az.ZoneName,
					HostName:         hostName,
					ServiceName:      serviceName,
					ServiceUpdatedAt: service.UpdatedAt,
				}
				if az.ZoneState.Available {
					azHost.ZoneState = "available"
				} else {
					azHost.ZoneState = "disabled"
				}
				if service.Active {
					azHost.ServiceStatus = "enabled"
				} else {
					azHost.ServiceStatus = "disabled"
				}
				if service.Available {
					azHost.ServiceAvailable = ":)"
				} else {
					azHost.ServiceAvailable = "XXX"
				}
				azHostList = append(azHostList, azHost)
			}
		}
	}

	pt := common.PrettyTable{
		ShortColumns: []common.Column{
			{Name: "ZoneName"}, {Name: "ZoneState", AutoColor: true}, {Name: "HostName"},
			{Name: "ServiceName"}, {Name: "ServiceStatus", AutoColor: true},
			{Name: "ServiceAvailable", AutoColor: true},
			{Name: "ServiceUpdatedAt", Text: "Updated At"},
		},
	}
	pt.AddItems(azHostList)
	common.PrintPrettyTable(pt, false)
}
func printAZInfoTree(azList []compute.AvailabilityZone) {
	tw := list.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetStyle(list.StyleConnectedRounded)

	for _, az := range azList {
		var zoneState string
		if az.ZoneState.Available {
			zoneState = common.BaseColorFormatter.Format("available")
		} else {
			zoneState = common.BaseColorFormatter.Format("disabled")
		}
		tw.AppendItem(fmt.Sprintf("%s %v", az.ZoneName, zoneState))
		tw.Indent()
		for hostName, services := range az.Hosts {
			tw.AppendItem(hostName)
			tw.Indent()
			for serviceName, service := range services {
				var (
					serviceStatus    string
					serviceAvailable string
				)
				if service.Active {
					serviceStatus = common.BaseColorFormatter.Format("enabled")
				} else {
					serviceStatus = common.BaseColorFormatter.Format("disabled")
				}
				if service.Available {
					serviceAvailable = common.BaseColorFormatter.Format(":)")
				} else {
					serviceAvailable = common.BaseColorFormatter.Format("XXX")
				}
				tw.AppendItem(
					fmt.Sprintf("%-20s %-10s %s", serviceName, serviceStatus, serviceAvailable),
				)
			}
			tw.UnIndent()
		}
		tw.UnIndent()
	}

	tw.Render()
}

func printAzInfoJson(azInfo []compute.AvailabilityZone) {
	jsonString, err := common.GetIndentJson(azInfo)
	if err != nil {
		logging.Fatal("get json string failed, %v", err)
	}
	fmt.Println(jsonString)
}

func printAzInfoYaml(azInfo []compute.AvailabilityZone) {
	yamlString, err := common.GetYaml(azInfo)
	if err != nil {
		logging.Fatal("get yaml string failed, %v", err)
	}
	fmt.Println(yamlString)
}
