package server

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/BytemanD/easygo/pkg/arrayutils"
	"github.com/BytemanD/easygo/pkg/global/logging"
	"github.com/BytemanD/easygo/pkg/syncutils"
	"github.com/BytemanD/skyman/openstack"
	"github.com/BytemanD/skyman/utility"
	"github.com/spf13/cobra"
)

var attachVolumes = &cobra.Command{
	Use:   "add-volumes <server>",
	Short: "Add volumes to server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serverId := args[0]

		nums, _ := cmd.Flags().GetInt("nums")
		parallel, _ := cmd.Flags().GetInt("parallel")
		size, _ := cmd.Flags().GetInt("size")
		volumeType, _ := cmd.Flags().GetString("type")

		client := openstack.DefaultClient()
		cinderClient := client.CinderV2()
		server, err := client.NovaV2().Servers().Show(serverId)
		utility.LogError(err, "show server failed:", true)

		volumes := []Volume{}
		mu := sync.Mutex{}

		taskGroup := syncutils.TaskGroup{
			Items:        arrayutils.Range(nums),
			MaxWorker:    parallel,
			ShowProgress: true,
			Func: func(item interface{}) error {
				p := item.(int)
				name := fmt.Sprintf("skyman-volume-%d", p+1)
				createOption := map[string]interface{}{
					"name": name, "size": size,
				}
				if volumeType != "" {
					createOption["volume_type"] = volumeType
				}
				logging.Debug("creating volume %s", name)
				volume, err := cinderClient.Volumes().CreateAndWait(createOption, 600)
				if err != nil {
					logging.Error("create volume failed: %v", err)
					return err
				}
				logging.Info("created volume: %v (%v)", volume.Name, volume.Id)
				mu.Lock()
				volumes = append(volumes, Volume{Id: volume.Id, Name: name})
				mu.Unlock()
				return nil
			},
		}
		logging.Info("creating %d volume(s), waiting ...", nums)
		taskGroup.Start()

		if len(volumes) == 0 {
			return
		}
		taskGroup2 := syncutils.TaskGroup{
			Items:        volumes,
			MaxWorker:    parallel,
			ShowProgress: true,
			Func: func(item interface{}) error {
				p := item.(Volume)
				logging.Debug("[volume: %s] attaching", p)
				attachment, err := client.NovaV2().Servers().AddVolume(server.Id, p.Id)
				if err != nil {
					logging.Error("[volume: %s] attach failed: %v", p, err)
					return err
				}
				if attachment != nil && p.Id == "" {
					p.Id = attachment.VolumeId
				}
				startTime := time.Now()
				for {
					attachedVolumes, err := client.NovaV2().Servers().ListVolumes(server.Id)
					if err != nil {
						utility.LogError(err, "list server volumes failed:", false)
						return err
					}
					for _, vol := range attachedVolumes {
						if vol.VolumeId != p.Id {
							continue
						}
						v, err := client.CinderV2().Volumes().Show(vol.VolumeId)
						logging.Info("[volume: %s] status is %s", vol.Id, v.Status)
						if err == nil && v.IsInuse() {
							logging.Info("[volume: %s] attach success", p.Id)
							return nil
						}
					}
					if time.Since(startTime) >= time.Minute*10 {
						break
					}
					time.Sleep(time.Second * 5)

				}
				logging.Error("[volume: %s] attach failed", p)
				return nil
			},
		}
		logging.Info("attaching ...")
		taskGroup2.Start()
	},
}

func init() {
	attachVolumes.Flags().Int("nums", 1, "nums of interfaces")
	attachVolumes.Flags().Int("parallel", runtime.NumCPU(), "nums of parallel")
	attachVolumes.Flags().Int("size", 10, "size of volume")
	attachVolumes.Flags().String("type", "", "attach volume with specified type")

	ServerCommand.AddCommand(attachVolumes)
}
