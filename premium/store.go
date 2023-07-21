package premium

import (
	"context"
	"os"

	"github.com/peterbourgon/diskv/v3"
)

type Servers struct {
	DataStore *diskv.Diskv
}

func (svrs *Servers) GetPremium(ctx context.Context) []string {
	servers := []string{}
	for k := range svrs.DataStore.Keys(ctx.Done()) {
		servers = append(servers, k)
	}
	return servers
}

func (svrs *Servers) IsPremium(GID string) bool {
	return svrs.DataStore.Has(GID)
}
func (svrs *Servers) Delete(GID string) error {
	err := svrs.DataStore.Erase(GID)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}
func (svrs *Servers) Add(GID string) error {
	jsonData := []byte("{}") // i imagine one day i might want to use the data stored about a premium guild
	err := svrs.DataStore.Write(GID, jsonData)
	return err
}
