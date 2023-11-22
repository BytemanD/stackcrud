package identity

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/BytemanD/easygo/pkg/global/logging"
	"github.com/BytemanD/skyman/openstack/common"
)

const (
	ContentType string = "application/json"

	TYPE_COMPUTE   string = "compute"
	TYPE_VOLUME    string = "volume"
	TYPE_VOLUME_V2 string = "volumev2"
	TYPE_VOLUME_V3 string = "volumev3"
	TYPE_IDENTITY  string = "identity"
	TYPE_IMAGE     string = "image"
	TYPE_NETWORK   string = "network"

	INTERFACE_PUBLIC   string = "public"
	INTERFACE_ADMIN    string = "admin"
	INTERFACE_INTERVAL string = "internal"

	URL_AUTH_TOKEN string = "/auth/tokens"
)

type V3AuthClient struct {
	restfulClient     common.RestfulClient
	AuthUrl           string
	Username          string
	Password          string
	ProjectName       string
	UserDomainName    string
	ProjectDomainName string
	RegionName        string

	tokenCache TokenCache
	session    common.Session
}

func (client *V3AuthClient) SetTimeout(timeout int) {
	client.session.Timeout = time.Second * time.Duration(timeout)
	client.restfulClient.Timeout = time.Second * time.Duration(timeout)
}

func (client *V3AuthClient) GetToken() Token {
	return client.tokenCache.token
}

func (client V3AuthClient) Request(req *http.Request) (*common.Response, error) {
	req.Header.Set("User-Agent", "go-skyman")
	if err := client.rejectToken(req); err != nil {
		return nil, err
	}
	return client.restfulClient.Request(req)
	// return client.session.Request(req)
}

func (client V3AuthClient) rejectToken(req *http.Request) error {
	tokenId, err := client.GetTokenId()
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Token", tokenId)
	return nil
}

func (client *V3AuthClient) GetTokenId() (string, error) {
	if client.isTokenExpired() {
		if err := client.TokenIssue(); err != nil {
			return "", err
		}
	}
	return client.tokenCache.TokenId, nil
}
func (client V3AuthClient) isTokenExpired() bool {
	if client.tokenCache.TokenId == "" {
		return true
	}
	if client.tokenCache.expiredAt.Before(time.Now()) {
		logging.Debug("token is exipred, expire second is %d", client.tokenCache.TokenExpireSecond)
		return true
	}
	return false
}
func (client *V3AuthClient) getAuthReqBody() AuthBody {
	authBody := AuthBody{}
	authBody.Auth.Identity.Methods = []string{"password"}

	authBody.Auth.Identity.Password.User.Name = client.Username
	authBody.Auth.Identity.Password.User.Password = client.Password
	authBody.Auth.Identity.Password.User.Domain.Name = client.UserDomainName
	authBody.Auth.Scope.Project.Name = client.ProjectName
	authBody.Auth.Scope.Project.Domain.Name = client.ProjectDomainName

	return authBody
}
func (client *V3AuthClient) TokenIssue() error {
	body, _ := json.Marshal(client.getAuthReqBody())
	resp, err := client.restfulClient.Post(
		fmt.Sprintf("%s%s", client.AuthUrl, URL_AUTH_TOKEN), body)
	if err != nil {
		return fmt.Errorf("token issue failed, %v", err)
	}
	var resToken RespToken
	resp.BodyUnmarshal(&resToken)

	client.tokenCache = TokenCache{
		token:     resToken.Token,
		TokenId:   resp.GetHeader("X-Subject-Token"),
		expiredAt: time.Now().Add(time.Second * time.Duration(client.tokenCache.TokenExpireSecond)),
	}
	return nil
}

func (client *V3AuthClient) SetTokenExpireSecond(second int) {
	client.tokenCache.TokenExpireSecond = second
}

func (client V3AuthClient) GetEndpointFromCatalog(serviceType string, endpointInterface string, region string) (string, error) {
	if len(client.tokenCache.token.Catalogs) == 0 {
		if err := client.TokenIssue(); err != nil {
			return "", err
		}
	}
	endpoints := client.tokenCache.GetEndpoints(OptionCatalog{
		Type:      serviceType,
		Interface: endpointInterface,
		Region:    region,
	})
	if (len(endpoints)) == 0 {
		return "", fmt.Errorf("endpoints not found")
	} else if strings.HasSuffix(endpoints[0].Url, "/") {
		return endpoints[0].Url[:len(endpoints[0].Url)-1], nil
	} else {
		return endpoints[0].Url, nil
	}
}

// 获取认证客户端
func GetV3AuthClient(authUrl string, user User, project Project, regionName string) (*V3AuthClient, error) {
	if authUrl == "" {
		return nil, fmt.Errorf("authUrl is missing")
	}

	client := V3AuthClient{
		AuthUrl:           authUrl,
		Username:          user.Name,
		Password:          user.Password,
		UserDomainName:    user.Domain.Name,
		ProjectName:       project.Name,
		ProjectDomainName: project.Domain.Name,
		RegionName:        regionName,
	}
	client.SetTokenExpireSecond(DEFAULT_TOKEN_EXPIRE_SECOND)
	if client.RegionName == "" {
		client.RegionName = "RegionOne"
	}
	return &client, nil
}
