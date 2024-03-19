package openstack

import (
	"fmt"

	"github.com/BytemanD/easygo/pkg/global/logging"
	"github.com/BytemanD/skyman/openstack/model/neutron"
	"github.com/BytemanD/skyman/utility"
)

func (o Openstack) PrunePorts(ports []neutron.Port) {
	c := o.NeutronV2()
	fmt.Printf("即将清理 %d 个Port(s):\n", len(ports))
	for _, port := range ports {
		fmt.Printf("%s (%s)\n", port.Id, port.Name)
	}
	yes := utility.ScanfComfirm("是否清理?", []string{"yes", "y"}, []string{"no", "n"})
	if !yes {
		return
	}
	tg := utility.TaskGroup{
		Func: func(i interface{}) error {
			port := i.(neutron.Port)
			logging.Debug("delete port %s(%s)", port.Id, port.Name)
			err := c.Ports().Delete(port.Id)
			if err != nil {
				return fmt.Errorf("delete port %s failed: %v", port.Id, err)
			}
			return nil
		},
		Items:        ports,
		ShowProgress: true,
	}
	err := tg.Start()
	if err != nil {
		logging.Error("清理失败: %v", err)
	} else {
		logging.Info("清理完成")
	}
}