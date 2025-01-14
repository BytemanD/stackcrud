package auth_plugin

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/BytemanD/go-console/console"
	"github.com/BytemanD/skyman/openstack/model"
	"github.com/BytemanD/skyman/openstack/session"
	"github.com/go-resty/resty/v2"
)

const (
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
	X_AUTH_TOKEN   string = "X-Auth-Token"
)

type PasswordAuthPlugin struct {
	AuthUrl           string
	Username          string
	Password          string
	ProjectName       string
	UserDomainName    string
	ProjectDomainName string
	RegionName        string

	LocalTokenExpireSecond int
	token                  *model.Token
	tokenId                string
	expiredAt              time.Time

	tokenLock *sync.Mutex

	mu      *sync.Mutex
	session *resty.Client
}

func (plugin PasswordAuthPlugin) Region() string {
	return plugin.RegionName
}

func (plugin *PasswordAuthPlugin) SetRegion(region string) {
	plugin.RegionName = region
}

func (plugin *PasswordAuthPlugin) SetLocalTokenExpire(expireSeconds int) {
	plugin.LocalTokenExpireSecond = expireSeconds
}

func (plugin *PasswordAuthPlugin) IsTokenExpired() bool {
	if plugin.tokenId == "" {
		return true
	}
	if plugin.expiredAt.Before(time.Now()) {
		console.Warn("token exipred, expired at: %s , now: %s", plugin.expiredAt, time.Now())
		return true
	}
	return false
}

func (plugin *PasswordAuthPlugin) makesureTokenValid() error {
	plugin.tokenLock.Lock()
	defer plugin.tokenLock.Unlock()

	if plugin.IsTokenExpired() {
		return plugin.TokenIssue()
	}
	return nil
}

func (plugin *PasswordAuthPlugin) GetToken() (*model.Token, error) {
	plugin.makesureTokenValid()
	return plugin.token, nil
}
func (plugin *PasswordAuthPlugin) GetTokenId() (string, error) {
	if err := plugin.makesureTokenValid(); err != nil {
		return "", err
	}
	return plugin.tokenId, nil
}

type AuthBody struct {
	Auth model.Auth `json:"auth"`
}

func (client PasswordAuthPlugin) newAuthReqBody() AuthBody {
	authData := model.Auth{
		Identity: model.Identity{
			Methods: []string{"password"},
			Password: model.Password{
				User: model.User{
					Name: client.Username, Password: client.Password,
					Domain: model.Domain{Name: client.UserDomainName}}},
		},
		Scope: model.Scope{Project: model.Project{
			Name:   client.ProjectName,
			Domain: model.Domain{Name: client.ProjectDomainName}},
		},
	}
	return AuthBody{Auth: authData}
}

func (plugin *PasswordAuthPlugin) TokenIssue() error {
	respBody := struct {
		Token model.Token `json:"token"`
	}{}
	resp, err := plugin.session.R().SetBody(plugin.newAuthReqBody()).
		SetResult(&respBody).
		Post(fmt.Sprintf("%s%s", plugin.AuthUrl, URL_AUTH_TOKEN))
	if err != nil || resp.Error() != nil {
		return fmt.Errorf("token issue failed, %s %s", err, resp.Error())
	}
	plugin.tokenId = resp.Header().Get("X-Subject-Token")
	plugin.token = &respBody.Token
	plugin.expiredAt = time.Now().Add(time.Second * time.Duration(plugin.LocalTokenExpireSecond))
	return nil
}
func (plugin *PasswordAuthPlugin) GetServiceEndpoints(sType string, sName string) ([]model.Endpoint, error) {
	token, err := plugin.GetToken()
	if err != nil {
		return nil, err
	}

	for _, catalog := range token.Catalogs {
		if catalog.Type != sType || (sName != "" && catalog.Name != sName) {
			continue
		}
		return catalog.Endpoints, nil
	}
	return []model.Endpoint{}, nil
}
func (plugin *PasswordAuthPlugin) GetServiceEndpoint(sType string, sName string, sInterface string) (string, error) {
	if err := plugin.makesureTokenValid(); err != nil {
		return "", fmt.Errorf("get catalogs failed: %s", err)
	}

	for _, catalog := range plugin.token.Catalogs {
		if catalog.Type != sType || (sName != "" && catalog.Name != sName) {
			continue
		}
		for _, endpoint := range catalog.Endpoints {
			if endpoint.Interface == sInterface && endpoint.Region == plugin.RegionName {
				return endpoint.Url, nil
			}
		}
	}
	return "", fmt.Errorf("endpoint %s:%s:%s for region '%s' not found",
		sType, sName, sInterface, plugin.RegionName)
}
func (plugin *PasswordAuthPlugin) SetHttpTimeout(timeout int) *PasswordAuthPlugin {
	plugin.session.SetTimeout(time.Second * time.Duration(timeout))
	return plugin
}
func (plugin *PasswordAuthPlugin) SetRetryWaitTime(waitTime int) *PasswordAuthPlugin {
	plugin.session.SetRetryWaitTime(time.Second * time.Duration(waitTime))
	return plugin
}
func (plugin *PasswordAuthPlugin) SetRetryCount(count int) *PasswordAuthPlugin {
	plugin.session.SetRetryCount(count)
	return plugin
}

func (plugin PasswordAuthPlugin) AuthRequest(req *resty.Request) error {
	plugin.mu.Lock()
	defer plugin.mu.Unlock()

	tokenId, err := plugin.GetTokenId()
	if err != nil {
		return err
	}
	if req.Header.Get(X_AUTH_TOKEN) == tokenId {
		return nil
	}
	console.Debug("set auth token %s", tokenId)
	req.Header.Set(X_AUTH_TOKEN, tokenId)
	return nil
}
func (plugin PasswordAuthPlugin) GetSafeHeader(header http.Header) http.Header {
	safeHeaders := http.Header{}
	for k, v := range header {
		if k == X_AUTH_TOKEN {
			safeHeaders[k] = []string{"<TOKEN>"}
		} else {
			safeHeaders[k] = v
		}
	}
	return safeHeaders
}
func (plugin PasswordAuthPlugin) GetProjectId() (string, error) {
	if err := plugin.makesureTokenValid(); err != nil {
		return "", err
	}
	return plugin.token.Project.Id, nil
}
func (plugin PasswordAuthPlugin) IsAdmin() bool {
	for _, role := range plugin.token.Roles {
		if role.Name == "admin" {
			return true
		}
	}
	return false
}

func NewPasswordAuthPlugin(authUrl string, user model.User, project model.Project, regionName string) *PasswordAuthPlugin {
	return &PasswordAuthPlugin{
		session:           session.DefaultRestyClient(),
		AuthUrl:           authUrl,
		Username:          user.Name,
		Password:          user.Password,
		UserDomainName:    user.Domain.Name,
		ProjectName:       project.Name,
		ProjectDomainName: project.Domain.Name,
		RegionName:        regionName,
		tokenLock:         &sync.Mutex{},
		mu:                &sync.Mutex{},
	}
}
