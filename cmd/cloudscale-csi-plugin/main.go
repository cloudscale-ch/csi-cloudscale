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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cloudscale-ch/csi-cloudscale/driver"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "unix:///var/lib/kubelet/plugins/"+driver.DriverName+"/csi.sock", "CSI endpoint")
		token    = flag.String("token", "", "cloudscale.ch access token")
		url      = flag.String("url", "https://api.cloudscale.ch/", "cloudscale.ch API URL")
		version  = flag.Bool("version", false, "Print the version and exit.")
	)
	flag.Parse()

	if *token == "" {
		*token = os.Getenv("CLOUDSCALE_ACCESS_TOKEN")
	}

	if *version {
		fmt.Printf("%s - %s (%s)\n", driver.GetVersion(), driver.GetCommit(), driver.GetTreeState())
		os.Exit(0)
	}

	drv, err := driver.NewDriver(*endpoint, *token, *url)
	if err != nil {
		log.Fatalln(err)
	}

	if err := drv.Run(); err != nil {
		log.Fatalln(err)
	}
}
