/*
Copyright cloudscale.ch
Copyright 2018 DigitalOcean

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cloudscale-ch/cloudscale-go-sdk"
	"github.com/kubernetes-csi/csi-test/pkg/sanity"
	"github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestDriverSuite(t *testing.T) {
	socket := "/tmp/csi.sock"
	endpoint := "unix://" + socket
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove unix domain socket file %s, error: %s", socket, err)
	}

	// fake cloudscale.ch Server, not working yet ...
	serverId := "987654"
	fake := &fakeAPI{
		t:       t,
		volumes: map[string]*cloudscale.Volume{},
		servers: map[string]*cloudscale.Server{
			serverId: &cloudscale.Server{},
		},
	}

	ts := httptest.NewServer(fake)
	defer ts.Close()

	cloudscaleClient := cloudscale.NewClient(nil)
	url, _ := url.Parse(ts.URL)
	cloudscaleClient.BaseURL = url

	driver := &Driver{
		endpoint:         endpoint,
		serverId:         serverId,
		region:           "zrh1",
		cloudscaleClient: cloudscaleClient,
		mounter:          &fakeMounter{},
		log:              logrus.New().WithField("test_enabed", true),
	}
	defer driver.Stop()

	go driver.Run()

	mntDir, err := ioutil.TempDir("", "mnt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(mntDir)

	mntStageDir, err := ioutil.TempDir("", "mnt-stage")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(mntStageDir)

	cfg := &sanity.Config{
		StagingPath:    mntStageDir,
		TargetPath:     mntDir,
		Address:        endpoint,
		TestVolumeSize: 50 * 1024 * 1024 * 1024,
	}

	sanity.Test(t, cfg)
}

// fakeAPI implements a fake, cached cloudscale.ch API
type fakeAPI struct {
	t       *testing.T
	volumes map[string]*cloudscale.Volume
	servers map[string]*cloudscale.Server
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// don't deal with servers for now
	if strings.HasPrefix(r.URL.Path, "/v1/servers/") {
		// for now we only do a GET, so we assume it's a GET and don't check
		// for the method
		id := filepath.Base(r.URL.Path)
		server, ok := f.servers[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
		} else {
			_ = json.NewEncoder(w).Encode(&server)
		}

		return
	}

	// rest is /v1/volumes related
	switch r.Method {
	case "GET":
		// A list call
		if !strings.HasPrefix(r.URL.String(), "/v1/volumes/") {
			volumes := []cloudscale.Volume{}
			if name := r.URL.Query().Get("name"); name != "" {
				for _, vol := range f.volumes {
					if vol.Name == name {
						volumes = append(volumes, *vol)
					}
				}
			} else {
				for _, vol := range f.volumes {
					volumes = append(volumes, *vol)
				}
			}

			err := json.NewEncoder(w).Encode(&volumes)
			if err != nil {
				f.t.Fatal(err)
			}
			return

		} else {
			// single volume get
			id := filepath.Base(r.URL.Path)
			vol, ok := f.volumes[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
			} else {
				_ = json.NewEncoder(w).Encode(&vol)
			}

			return
		}

		// response with zero items
		err := json.NewEncoder(w).Encode(&[]*cloudscale.Volume{})
		if err != nil {
			f.t.Fatal(err)
		}
	case "POST":
		v := new(cloudscale.Volume)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			f.t.Fatal(err)
		}

		id := randString(30)
		vol := &cloudscale.Volume{
			UUID:   id,
			Name:   v.Name,
			SizeGB: v.SizeGB,
			Type:   v.Type,
		}

		f.volumes[id] = vol

		err = json.NewEncoder(w).Encode(&vol)
		if err != nil {
			f.t.Fatal(err)
		}
	case "DELETE":
		id := filepath.Base(r.URL.Path)
		delete(f.volumes, id)
	case "PATCH":
		v := new(cloudscale.Volume)
		err := json.NewDecoder(r.Body).Decode(v)
		if err != nil {
			f.t.Fatal(err)
		}
		// Check if volume exists
		id := filepath.Base(r.URL.Path)
		_, ok := f.volumes[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
		}

		// Check if server exists
		for _, serverUUID := range *v.ServerUUIDs {
			_, ok := f.servers[serverUUID]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	default:
		f.t.Fatalf("Used the unhandled HTTP method %s\n", r.Method)
	}
}

func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

type fakeMounter struct{}

func (f *fakeMounter) Format(source string, fsType string, luksContext LuksContext) error {
	return nil
}

func (f *fakeMounter) Mount(source string, target string, fsType string, luksContext LuksContext, options ...string) error {
	return nil
}

func (f *fakeMounter) Unmount(target string, luksContext LuksContext) error {
	return nil
}

func (f *fakeMounter) IsFormatted(source string, luksContext LuksContext) (bool, error) {
	return true, nil
}
func (f *fakeMounter) IsMounted(target string) (bool, error) {
	return true, nil
}
