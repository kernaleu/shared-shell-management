/*
 * A program for managing users and resource limits on kernal.eu servers.
 * Copyright (C) 2022 İrem Kuyucu <siren@kernal.eu>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.

 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.

 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"context"
	"log"
	"os"

	"gopkg.in/yaml.v3"
	"github.com/coreos/go-systemd/v22/dbus"
)

const shell = "/bin/bash"

type systemUser struct {
	Username string `yaml:"username"`
	Id string `yaml:"id"`
	PublicKey string `yaml:"public_key"`
	SystemdLimits string `yaml:"systemd_limits"`
	State string `yaml:"state"`
}

func main() {
	systemd, err := dbus.NewSystemdConnectionContext(context.Background())
	if err != nil {
		log.Fatal("Failed to connect to systemd:", err)
	}

	ub, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	var users []systemUser
	if err := yaml.Unmarshal(ub, &users); err != nil {
		log.Fatal(err)
	}

	for _, u := range users {
		switch(u.State) {
		case "present":
			createUser(u)
		}
	}

	if err := systemd.ReloadContext(context.Background()); err != nil {
		log.Fatal("Failed to reload daemon:", err)
	}
}
