package neutron

import (
	"fmt"

	"github.com/BytemanD/easygo/pkg/global/logging"
	"github.com/BytemanD/easygo/pkg/stringutils"

	"github.com/BytemanD/skyman/openstack"
	"github.com/BytemanD/skyman/utility"
	"github.com/spf13/cobra"
)

var Vpc = &cobra.Command{Use: "vpc"}

var vpcCreate = &cobra.Command{
	Use:   "create <name> <cidr>",
	Short: "Create VPC",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		vpc, cidr := args[0], args[1]
		c := openstack.DefaultClient().NeutronV2()
		routerName := fmt.Sprintf("%s-router", vpc)
		networkName := fmt.Sprintf("%s-network", vpc)
		subnetName := fmt.Sprintf("%s-subnet", vpc)

		// create router
		routerParams := map[string]interface{}{"name": routerName}
		logging.Info("create router %s", routerName)
		router, err := c.Router().Create(routerParams)
		utility.LogIfError(err, true, "create router %s failed", routerName)
		// create network
		networkParams := map[string]interface{}{"name": networkName}
		logging.Info("create network %s", networkName)
		network, err := c.Network().Create(networkParams)
		utility.LogIfError(err, true, "create network %s failed", networkParams)
		// create router
		subnetParams := map[string]interface{}{
			"name": subnetName, "network_id": network.Id,
			"cidr": cidr, "ip_version": "4",
		}
		logging.Info("create subnet %s", subnetName)
		subnet, err := c.Subnet().Create(subnetParams)
		utility.LogIfError(err, true, "create subnet %s failed", subnetName)
		// add router interface
		logging.Info("add subnet %s to router %s", subnetName, routerName)
		err = c.Router().AddSubnet(router.Id, subnet.Id)
		utility.LogIfError(err, true, "add subnet %s to router %s failed", subnetName, routerName)
		logging.Info("create VPC %s success", vpc)
	},
}
var vpcDelete = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete VPC",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		vpc := args[0]
		routerName, _ := cmd.Flags().GetString("router-name")

		if routerName == "" {
			routerName = fmt.Sprintf("%s-router", vpc)
		}

		c := openstack.DefaultClient().NeutronV2()
		// get vpc router
		logging.Info("get router %s", routerName)
		router, err := c.Router().Found(routerName)
		utility.LogIfError(err, true, "get router %s failed", routerName)
		// remove router ports
		routerPorts, err := c.ListRouterPorts(router.Id)
		utility.LogIfError(err, true, "list router ports failed")
		subnets := []string{}
		for _, port := range routerPorts {
			for _, fixedIp := range port.FixedIps {
				logging.Info("remove subnet %s from router %s", fixedIp.SubnetId, router.Id)
				c.Router().RemoveSubnet(router.Id, fixedIp.SubnetId)
				if !stringutils.ContainsString(subnets, fixedIp.SubnetId) {
					subnets = append(subnets, fixedIp.SubnetId)
				}
			}
		}
		// delete vpc networks
		for _, subnetId := range subnets {
			subnet, err := c.Subnet().Show(subnetId)
			utility.LogIfError(err, true, "get subnet %s failed", subnetId)
			logging.Info("delete vpc network %s", subnet.NetworkId)
			err = c.Network().Delete(subnet.NetworkId)
			utility.LogIfError(err, true, "delete network %s failed", subnet.NetworkId)
		}

		// delete vpc router
		logging.Info("delete vpc router %s", routerName)
		err = c.Router().Delete(router.Id)
		utility.LogIfError(err, true, "dele router %s failed", routerName)
		logging.Info("VPC %s delete success", vpc)
	},
}

func init() {
	vpcDelete.Flags().StringP("router-name", "n", "", "router name")

	Vpc.AddCommand(vpcCreate, vpcDelete)
}
