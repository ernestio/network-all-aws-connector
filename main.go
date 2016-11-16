/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"fmt"
	"os"
	"runtime"

	ecc "github.com/ernestio/ernest-config-client"
	"github.com/ernestio/ernestaws"
	"github.com/ernestio/ernestaws/network"
	"github.com/nats-io/nats"
)

var nc *nats.Conn
var natsErr error
var err error

func eventHandler(m *nats.Msg) {
	n := network.New(m.Subject, m.Data)

	subject, data := ernestaws.Handle(&n)
	nc.Publish(subject, data)
}

func main() {
	nc = ecc.NewConfig(os.Getenv("NATS_URI")).Nats()

	events := []string{"network.create.aws", "network.delete.aws"}
	for _, subject := range events {
		fmt.Println("listening for " + subject)
		nc.Subscribe(subject, eventHandler)
	}

	runtime.Goexit()
}
