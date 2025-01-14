package internal

import (
	"fmt"

	"github.com/BytemanD/go-console/console"
)

type ServerRebuild struct {
	ServerActionTest
	EmptyCleanup
}

func (t ServerRebuild) Start() error {
	err := t.Client.NovaV2().Server().Rebuild(t.Server.Id, map[string]interface{}{})
	if err != nil {
		return err
	}
	console.Info("[%s] rebuilding", t.Server.Id)
	if err := t.WaitServerTaskFinished(false); err != nil {
		return err
	}
	if t.Server.IsError() {
		return fmt.Errorf("server is error")
	}
	return nil
}
