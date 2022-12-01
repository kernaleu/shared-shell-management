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
	"os/user"
	"os/exec"
	"errors"

	"gopkg.in/yaml.v3"
	"github.com/coreos/go-systemd/v22/dbus"
)

const shell = "/bin/bash"

type systemUser struct {
	Username string `yaml:"username"`
	Id string `yaml:"id"`
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

	userDoesNotExist := new(user.UnknownUserError)
	for _, u := range users {
		if _, err := user.Lookup(u.Username); !errors.As(err, userDoesNotExist) {
			if err != nil {
				log.Fatal(err)
			}
			// Skip pre existing user
			continue
		}

		bdir := "/home/" + u.Username[0:1]
		// Create base directory
		if _, err := os.Stat(bdir); errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(bdir, 0755); err != nil {
				log.Fatal(err)
			}
		}

		// TODO: There isn't a library that does this programatically, yet.
		if err := exec.Command("groupadd", "-g", u.Id, u.Username).Run(); err != nil {
			log.Fatalf("Failed to create group %s: %s", u.Username, err.Error())
		}
		if err := exec.Command("useradd", "-b", bdir, "-m", "-u", u.Id, "-g", u.Username,
		    "-s", shell, u.Username).Run(); err != nil {
			log.Fatalf("Failed to create user %s: %s", u.Username, err.Error())
		}

		// Create systemd user slice
		sliceDir := "/etc/systemd/system/user-" + u.Id + ".slice.d"
		if err := os.Mkdir(sliceDir, 0755); err != nil {
			log.Fatal("Failed to create directory for user slice:", err)
		}
		f, err := os.OpenFile(sliceDir + "/override.conf", os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("Failed to create user slice override.conf:", err)
		}
		f.WriteString(u.SystemdLimits)
		f.Close()
	}

	if err := systemd.ReloadContext(context.Background()); err != nil {
		log.Fatal("Failed to reload daemon:", err)
	}
}
