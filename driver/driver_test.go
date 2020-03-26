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
	"math/rand"
	"os"
	"strconv"
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

	serverId := 987654

	cloudscaleClient := NewFakeClient()
	driver := &Driver{
		endpoint:         endpoint,
		serverId:         strconv.Itoa(serverId),
		region:           "zrh1",
		cloudscaleClient: cloudscaleClient,
		mounter:          &fakeMounter{},
		log:              logrus.New().WithField("test_enabed", true),
	}
	defer driver.Stop()

	go driver.Run()

	cfg := &sanity.Config{
		Address:        endpoint,
		TestVolumeSize: 50 * 1024 * 1024 * 1024,
	}

	sanity.Test(t, cfg)
}

func NewFakeClient() *cloudscale.Client {
	userAgent := "cloudscale/" + "fake"
	fakeClient := &cloudscale.Client{BaseURL: nil, UserAgent: userAgent}
	return fakeClient
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
func (f *fakeMounter) FinalizeVolumeAttachmentAndFindPath(logger *logrus.Entry, target string) (*string, error) {
	path := "SomePath"
	return &path, nil
}

func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
