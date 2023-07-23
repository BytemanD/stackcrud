package compute

import (
	"encoding/json"
	"fmt"
	"net/url"
	netUrl "net/url"
	"strings"
	"time"

	"github.com/BytemanD/easygo/pkg/global/logging"
)

func (client ComputeClientV2) ServerList(query netUrl.Values) Servers {
	serversBody := ServersBody{}
	client.List("servers", query, client.BaseHeaders, &serversBody)
	return serversBody.Servers
}
func (client ComputeClientV2) ServerListDetails(query netUrl.Values) Servers {
	serversBody := ServersBody{}
	client.List("servers/detail", query, client.BaseHeaders, &serversBody)
	return serversBody.Servers
}
func (client ComputeClientV2) ServerListDetailsByName(name string) Servers {
	query := url.Values{}
	query.Set("name", name)
	return client.ServerListDetails(query)
}
func (client ComputeClientV2) ServerShow(id string) (*Server, error) {
	serverBody := ServerBody{}
	err := client.Show("servers", id, client.BaseHeaders, &serverBody)
	return serverBody.Server, err
}

func (client ComputeClientV2) ServerDelete(id string) error {
	return client.Delete("servers", id, client.BaseHeaders)
}
func (client ComputeClientV2) getBlockDeviceMappingV2(imageRef string) BlockDeviceMappingV2 {
	return BlockDeviceMappingV2{
		BootIndex:          0,
		UUID:               imageRef,
		VolumeSize:         10,
		SourceType:         "image",
		DestinationType:    "volume",
		DeleteOnTemination: true,
	}
}
func (client ComputeClientV2) ServerCreate(options ServerOpt) (*Server, error) {
	if options.Name == "" {
		return nil, fmt.Errorf("name is empty")
	}
	if options.Flavor == "" {
		return nil, fmt.Errorf("flavor is empty")
	}
	if options.Image == "" ||
		(len(options.BlockDeviceMappingV2) > 0 && options.BlockDeviceMappingV2[0].UUID == "") {
		return nil, fmt.Errorf("image is empty")
	}

	if options.Networks == nil {
		options.Networks = "none"
	}

	body, _ := json.Marshal(ServeCreaterBody{Server: options})
	serverBody := ServerBody{}
	var createErr error
	if options.BlockDeviceMappingV2 != nil {
		createErr = client.Create("os-volumes_boot", body, client.BaseHeaders, &serverBody)
	} else {
		createErr = client.Create("servers", body, client.BaseHeaders, &serverBody)
	}
	return serverBody.Server, createErr
}
func (client ComputeClientV2) WaitServerCreate(options ServerOpt) (*Server, error) {
	server, err := client.ServerCreate(options)
	if err != nil {
		return server, err
	}
	if server.Id == "" {
		return server, fmt.Errorf("create server failed")
	}
	return client.WaitServerStatusSecond(server.Id, "ACTIVE", 5)
}

func (client ComputeClientV2) WaitServerStatusSecond(serverId string, status string, second int) (*Server, error) {
	// var server Server
	for {
		server, err := client.ServerShow(serverId)
		if err != nil {
			return server, err
		}
		logging.Debug("server stauts is %s", server.Status)
		switch strings.ToUpper(server.Status) {
		case "ERROR":
			return server, fmt.Errorf("server status is error, message: %s", server.Fault.Message)
		case strings.ToUpper(status):
			return server, nil
		}
		time.Sleep(time.Second * time.Duration(second))
	}
}

func (client ComputeClientV2) WaitServerStatus(serverId string, status string) (*Server, error) {
	return client.WaitServerStatusSecond(serverId, status, 1)
}

func (client ComputeClientV2) WaitServerDeleted(id string) {
	client.ServerDelete(id)
	for {
		server, err := client.ServerShow(id)
		if server.Id == "" || err != nil {
			break
		}
		logging.Debug("servers status is %s", server.Status)
		time.Sleep(time.Second * 2)
	}
}

// service api
func (client ComputeClientV2) ServiceList(query netUrl.Values) Services {
	body := ServicesBody{}
	client.List("os-services", query, client.BaseHeaders, &body)
	return body.Services
}
func (client ComputeClientV2) ServiceAction(action string, host string, binary string) *Service {
	req := Service{Binary: binary, Host: host}
	reqBody, _ := json.Marshal(req)
	body := map[string]Service{"service": {}}
	client.Put("os-services/"+action, reqBody, client.BaseHeaders, &body)
	service := body["service"]
	return &service
}
func (client ComputeClientV2) ServiceUpdate(id string, update map[string]interface{}) (*Service, error) {
	reqBody, _ := json.Marshal(update)
	body := map[string]Service{"service": {}}

	if err := client.Put("os-services/"+id, reqBody, client.BaseHeaders, &body); err != nil {
		return nil, err
	}
	reqService := body["service"]
	return &reqService, nil
}
func (client ComputeClientV2) ServiceGetByHostBinary(host string, binary string) (*Service, error) {
	query := netUrl.Values{"host": []string{host}, "binary": []string{binary}}
	services := client.ServiceList(query)
	if len(services) == 0 {
		return nil, fmt.Errorf("service %s:%s not found", host, binary)
	}
	return &services[0], nil
}
func (client ComputeClientV2) ServiceUp(host string, binary string) (*Service, error) {
	if client.MicroVersionLargeThen(2.53) {
		service, err := client.ServiceGetByHostBinary(host, binary)
		if err != nil {
			return nil, err
		}
		return client.ServiceUpdate(service.Id,
			map[string]interface{}{"forced_down": false})
	}
	return client.ServiceAction("up", host, binary), nil
}
func (client ComputeClientV2) ServiceDown(host string, binary string) (*Service, error) {
	if client.MicroVersionLargeThen(2.53) {
		service, err := client.ServiceGetByHostBinary(host, binary)
		if err != nil {
			return nil, err
		}
		return client.ServiceUpdate(service.Id, map[string]interface{}{"forced_down": true})
	}
	return client.ServiceAction("down", host, binary), nil
}
func (client ComputeClientV2) ServiceEnable(host string, binary string) (*Service, error) {
	if client.MicroVersionLargeThen(2.53) {
		service, err := client.ServiceGetByHostBinary(host, binary)
		if err != nil {
			return nil, err
		}
		return client.ServiceUpdate(service.Id, map[string]interface{}{"status": "enabled"})
	}
	return client.ServiceAction("enable", host, binary), nil
}
func (client ComputeClientV2) ServiceDisable(host string, binary string,
	reason string,
) (*Service, error) {
	if client.MicroVersionLargeThen(2.53) {
		service, err := client.ServiceGetByHostBinary(host, binary)
		if err != nil {
			return nil, err
		}
		body := map[string]interface{}{"status": "disabled"}
		if reason != "" {
			body["disabled_reason"] = reason
		}
		return client.ServiceUpdate(service.Id, body)
	}
	return client.ServiceAction("disable", host, binary), nil
}

// server api
func (client ComputeClientV2) ServerAction(action string, id string,
	params interface{},
) error {
	actionBody := map[string]interface{}{action: params}
	body, _ := json.Marshal(actionBody)
	err := client.Create(fmt.Sprintf("%s/%s/action", "servers", id), body,
		client.BaseHeaders, nil)
	return err
}
func (client ComputeClientV2) ServerStop(id string) error {
	return client.ServerAction("os-stop", id, nil)
}
func (client ComputeClientV2) ServerStart(id string) error {
	return client.ServerAction("os-start", id, nil)
}
func (client ComputeClientV2) ServerReboot(id string, hard bool) error {
	actionBody := map[string]string{}
	if hard {
		actionBody["type"] = "HARD"
	} else {
		actionBody["type"] = "SOFT"
	}
	return client.ServerAction("reboot", id, actionBody)
}
func (client ComputeClientV2) ServerPause(id string) error {
	return client.ServerAction("pause", id, nil)
}
func (client ComputeClientV2) ServerUnpause(id string) error {
	return client.ServerAction("unpause", id, nil)
}
func (client ComputeClientV2) ServerShelve(id string) error {
	return client.ServerAction("unshelve", id, nil)
}
func (client ComputeClientV2) ServerUnshelve(id string) error {
	return client.ServerAction("unshelve", id, nil)
}
func (client ComputeClientV2) ServerSuspend(id string) error {
	return client.ServerAction("suspend", id, nil)
}
func (client ComputeClientV2) ServerResume(id string) error {
	return client.ServerAction("resume", id, nil)
}

// server action api
func (client ComputeClientV2) ServerActionList(id string) ([]InstanceAction, error) {
	body := map[string][]InstanceAction{"instanceActions": {}}
	err := client.List(fmt.Sprintf("servers/%s/os-instance-actions", id), nil, client.BaseHeaders, &body)
	if err != nil {
		return nil, err
	}
	return body["instanceActions"], nil
}
func (client ComputeClientV2) ServerActionShow(id string, requestId string) (
	*InstanceAction, error,
) {
	body := map[string]InstanceAction{"instanceAction": {}}
	err := client.List(fmt.Sprintf("servers/%s/os-instance-actions/%s", id, requestId),
		nil, client.BaseHeaders, &body)
	if err != nil {
		return nil, err
	}
	instanceAction := body["instanceAction"]
	return &instanceAction, nil
}

// flavor api
func (client ComputeClientV2) FlavorList(query netUrl.Values) (Flavors, error) {
	body := FlavorsBody{}
	client.List("flavors", query, client.BaseHeaders, &body)
	return body.Flavors, nil
}
func (client ComputeClientV2) FlavorListDetail(query netUrl.Values) (Flavors, error) {
	body := map[string]Flavors{"flavors": {}}
	err := client.List("flavors/detail", query, client.BaseHeaders, &body)
	if err != nil {
		return nil, err
	}
	return body["flavors"], nil
}
func (client ComputeClientV2) FlavorExtraSpecsList(flavorId string) (ExtraSpecs, error) {
	body := ExtraSpecsBody{}
	err := client.List(
		fmt.Sprintf("flavors/%s/os-extra_specs", flavorId), nil, client.BaseHeaders,
		&body)
	if err != nil {
		return nil, err
	}
	return body.ExtraSpecs, nil
}

// hypervisor api
func (client ComputeClientV2) HypervisorList(query netUrl.Values) (Hypervisors, error) {
	body := map[string]Hypervisors{
		"hypervisors": []Hypervisor{},
	}
	client.List("os-hypervisors", query, client.BaseHeaders, &body)
	return body["hypervisors"], nil
}
func (client ComputeClientV2) HypervisorListDetail(query netUrl.Values) (Hypervisors, error) {
	body := map[string]Hypervisors{"hypervisors": {}}
	err := client.List("os-hypervisors/detail", query, client.BaseHeaders, &body)
	if err != nil {
		return nil, err
	}
	return body["hypervisors"], nil
}

// keypair api
func (client ComputeClientV2) KeypairList(query netUrl.Values) ([]Keypair, error) {
	body := map[string][]Keypair{"keypairs": {}}
	err := client.List("os-keypairs", query, client.BaseHeaders, &body)
	if err != nil {
		return nil, err
	}
	return body["keypairs"], nil
}

// server volumes api

func (client ComputeClientV2) ServerVolumeList(id string) ([]VolumeAttachment, error) {
	body := map[string][]VolumeAttachment{"volumeAttachments": {}}
	err := client.List(
		fmt.Sprintf("servers/%s/os-volume_attachments", id),
		nil, client.BaseHeaders, &body)
	if err != nil {
		return nil, err
	}
	return body["volumeAttachments"], nil
}
func (client ComputeClientV2) ServerVolumeAdd(id string, volumeId string) (*VolumeAttachment, error) {
	data := map[string]map[string]string{
		"volumeAttachment": {"volumeId": volumeId}}
	reqBody, _ := json.Marshal(data)
	respBody := map[string]*VolumeAttachment{"volumeAttachment": {}}
	err := client.Create(fmt.Sprintf("servers/%s/os-volume_attachments", id),
		reqBody, client.BaseHeaders, &respBody)
	if err != nil {
		return nil, err
	}
	return respBody["volumeAttachment"], nil
}
func (client ComputeClientV2) ServerVolumeDelete(id string, volumeId string) error {
	return client.Delete(
		fmt.Sprintf("servers/%s/os-volume_attachments", id),
		volumeId, client.BaseHeaders)

}
func (client ComputeClientV2) ServerInterfaceList(id string) ([]InterfaceAttachment, error) {
	body := map[string][]InterfaceAttachment{"interfaceAttachments": {}}
	err := client.List(
		fmt.Sprintf("servers/%s/os-interface", id),
		nil, client.BaseHeaders, &body)
	if err != nil {
		return nil, err
	}
	return body["interfaceAttachments"], nil
}

func (client ComputeClientV2) ServerAddNet(id string, netId string) (*InterfaceAttachment, error) {
	data := map[string]string{"net_id": netId}
	reqBody, _ := json.Marshal(map[string]interface{}{"interfaceAttachment": data})

	body := map[string]*InterfaceAttachment{"interfaceAttachment": {}}
	err := client.Create(fmt.Sprintf("servers/%s/os-interface", id),
		reqBody, client.BaseHeaders, &body)
	if err != nil {
		return nil, err
	}
	return body["interfaceAttachment"], nil
}
func (client ComputeClientV2) ServerAddPort(id string, portId string) (*InterfaceAttachment, error) {
	data := map[string]string{"port_id": portId}
	reqBody, _ := json.Marshal(map[string]interface{}{"interfaceAttachment": data})

	body := map[string]*InterfaceAttachment{"interfaceAttachment": {}}
	err := client.Create(fmt.Sprintf("servers/%s/os-interface", id),
		reqBody, client.BaseHeaders, &body)
	if err != nil {
		return nil, err
	}
	return body["interfaceAttachment"], nil
}

func (client ComputeClientV2) ServerInterfaceDetach(id string, portId string) error {
	return client.Delete(
		fmt.Sprintf("servers/%s/os-interface", id), portId, client.BaseHeaders)
}
