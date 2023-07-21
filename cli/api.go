package cli

import (
	"fmt"

	"github.com/BytemanD/stackcrud/common"
	"github.com/BytemanD/stackcrud/openstack"
	"github.com/BytemanD/stackcrud/openstack/identity"
)

func getAuthClient() (*identity.V3AuthClient, error) {
	authClient, err := identity.GetV3AuthClient(
		common.CONF.Auth.Url, common.CONF.Auth.User,
		common.CONF.Auth.Project, common.CONF.Auth.RegionName,
	)
	if err != nil {
		return nil, fmt.Errorf("获取认证客户端失败, %s", err)
	}
	if err := authClient.TokenIssue(); err != nil {
		return nil, fmt.Errorf("获取 Token 失败, %s", err)
	}
	return authClient, nil
}

func GetClient() (*openstack.OpenstackClient, error) {
	authClient, err := getAuthClient()
	if err != nil {
		return nil, err
	}
	client, err := openstack.GetClientWithAuthToken(authClient)
	if err == nil {
		client.Compute.UpdateVersion()
	}
	return client, err
}