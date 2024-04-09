package templates

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/BytemanD/easygo/pkg/global/logging"
	"github.com/BytemanD/skyman/openstack"
	"github.com/BytemanD/skyman/openstack/model"
	"github.com/BytemanD/skyman/openstack/model/glance"
	"github.com/BytemanD/skyman/openstack/model/neutron"
	"github.com/BytemanD/skyman/openstack/model/nova"
	"github.com/BytemanD/skyman/utility"
)

func getImage(client *openstack.Openstack, resource BaseResource) (*glance.Image, error) {
	if resource.Id != "" {
		logging.Info("find image %s", resource.Id)
		return client.GlanceV2().Images().Show(resource.Id)
	} else if resource.Name != "" {
		logging.Info("find image %s", resource.Name)
		return client.GlanceV2().Images().Found(resource.Name)
	} else {
		return nil, fmt.Errorf("image is empty")
	}
}
func createFlavor(client *openstack.Openstack, flavor Flavor) {
	computeClient := client.NovaV2()
	f, _ := computeClient.Flavors().Show(flavor.Id)
	if f != nil {
		logging.Warning("network %s exists", flavor.Id)
		return
	}
	newFlavor := nova.Flavor{
		Id:    flavor.Id,
		Name:  flavor.Name,
		Vcpus: flavor.Vcpus,
		Ram:   flavor.Ram,
	}
	logging.Info("creating flavor %s", newFlavor.Id)
	f, err := computeClient.Flavors().Create(newFlavor)
	utility.LogError(err, "create flavor failed", true)
	if flavor.ExtraSpecs != nil {
		logging.Info("creating flavor extra specs")
		_, err = computeClient.Flavors().SetExtraSpecs(f.Id, flavor.ExtraSpecs)
		utility.LogError(err, "create flavor extra specs failed", true)
	}
}
func createNetwork(client *openstack.Openstack, network Network) {
	networkClient := client.NeutronV2()
	_, err := networkClient.Networks().Found(network.Name)
	if err == nil {
		logging.Warning("network %s exists", network.Name)
		return
	}
	netParams := map[string]interface{}{
		"name": network.Name,
	}
	logging.Info("creating network %s", network.Name)
	net, err := networkClient.Networks().Create(netParams)
	utility.LogError(err, fmt.Sprintf("create network %s failed", network.Name), true)
	for _, subnet := range network.Subnets {
		if subnet.IpVersion == 0 {
			subnet.IpVersion = 4
		}
		subnetParams := map[string]interface{}{
			"name":       subnet.Name,
			"network_id": net.Id,
			"cidr":       subnet.Cidr,
			"ip_version": subnet.IpVersion,
		}
		logging.Info("creating subnet %s (cidr: %s)", subnet.Name, subnet.Cidr)
		_, err2 := networkClient.Subnets().Create(subnetParams)
		utility.LogError(err2, fmt.Sprintf("create subnet %s failed", subnet.Name), true)
	}
}

func createServer(client *openstack.Openstack, server Server, watch bool) (*nova.Server, error) {
	computeClient := client.NovaV2()
	networkClient := client.NeutronV2()

	s, _ := client.NovaV2().Servers().Found(server.Name)
	if s != nil {
		logging.Warning("server %s exists", s.Name)
		return s, nil
	}
	serverOption := nova.ServerOpt{
		Name:             server.Name,
		AvailabilityZone: server.AvailabilityZone,
		MinCount:         server.Min,
		MaxCount:         server.Max,
	}
	for _, sg := range server.SecurityGroups {
		serverOption.SecurityGroups = append(
			serverOption.SecurityGroups,
			neutron.SecurityGroup{
				Resource: model.Resource{Name: sg.Name},
			})
	}

	var flavor *nova.Flavor
	var err error
	if server.Flavor.Id == "" && server.Flavor.Name == "" {
		return nil, fmt.Errorf("flavor is empty")
	}

	if server.Flavor.Id != "" {
		logging.Info("find flavor %s", server.Flavor.Id)
		flavor, err = client.NovaV2().Flavors().Show(server.Flavor.Id)
	} else if server.Flavor.Name != "" {
		logging.Info("find flavor %s", server.Flavor.Name)
		flavor, err = client.NovaV2().Flavors().Found(server.Flavor.Name)
	}
	utility.LogError(err, "get flavor failed", true)
	serverOption.Flavor = flavor.Id

	if server.Image.Id != "" || server.Image.Name != "" {
		img, err := getImage(client, server.Image)
		utility.LogError(err, "get image failed", true)
		serverOption.Image = img.Id
	}

	if len(server.BlockDeviceMappingV2) > 0 {
		serverOption.BlockDeviceMappingV2 = []nova.BlockDeviceMappingV2{}
		for _, bdm := range server.BlockDeviceMappingV2 {
			serverOption.BlockDeviceMappingV2 = append(serverOption.BlockDeviceMappingV2,
				nova.BlockDeviceMappingV2{
					BootIndex:          bdm.BootIndex,
					UUID:               bdm.UUID,
					VolumeSize:         bdm.VolumeSize,
					VolumeType:         bdm.VolumeType,
					SourceType:         bdm.SourceType,
					DestinationType:    bdm.DestinationType,
					DeleteOnTemination: bdm.DeleteOnTermination,
				},
			)
		}
	}
	if len(server.Networks) > 0 {
		networks := []nova.ServerOptNetwork{}
		for _, nic := range server.Networks {
			if nic.UUID != "" {
				networks = append(networks, nova.ServerOptNetwork{UUID: nic.UUID})
			} else if nic.Port != "" {
				networks = append(networks, nova.ServerOptNetwork{Port: nic.Port})
			} else if nic.Name != "" {
				network, err := networkClient.Networks().Found(nic.Name)
				utility.LogError(err, "found network failed", true)
				networks = append(networks, nova.ServerOptNetwork{UUID: network.Id})
			}
		}
		serverOption.Networks = networks
	}
	if server.UserData != "" {
		content, err := utility.LoadUserData(server.UserData)
		utility.LogError(err, "read user data failed", true)
		serverOption.UserData = content
	}
	s, err = computeClient.Servers().Create(serverOption)
	utility.LogError(err, "create server failed", true)
	logging.Info("creating server %s", serverOption.Name)
	if watch {
		computeClient.Servers().WaitStatus(s.Id, "ACTIVE", 5)
	}
	return s, nil
}

var CreateCmd = &cobra.Command{
	Use:   "create <file>",
	Short: "create from template file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		watch, _ := cmd.Flags().GetBool("wait")
		var err error
		createTemplate, err := LoadCreateTemplate(args[0])
		utility.LogError(err, "load template file failed", true)

		for _, server := range createTemplate.Servers {
			if server.Name == "" {
				logging.Fatal("invalid config, server name is empty")
			}
			if server.Flavor.Id == "" && server.Flavor.Name == "" {
				logging.Fatal("invalid config, server flavor is empty")
			}
			if server.Image.Id == "" && server.Image.Name == "" && len(server.BlockDeviceMappingV2) == 0 {
				logging.Fatal("invalid config, server image is empty")
			}
		}
		client := openstack.DefaultClient()
		for _, flavor := range createTemplate.Flavors {
			createFlavor(client, flavor)
		}
		for _, network := range createTemplate.Networks {
			createNetwork(client, network)
		}

		for _, server := range createTemplate.Servers {
			_, err := createServer(client, server, watch)
			utility.LogError(err, "create server failed", true)
		}
	},
}

func init() {
	CreateCmd.Flags().Bool("wait", false, "wait the resource progress until it created.")
}
