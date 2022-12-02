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
	"strconv"

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

func createUser(u systemUser) {
	userDoesNotExist := new(user.UnknownUserError)
	if _, err := user.Lookup(u.Username); !errors.As(err, userDoesNotExist) {
		if err != nil {
			log.Fatal(err)
		}
		// Skip pre existing user
		return
	}

	homeDir := "/home/" + u.Username[0:1] + "/" + u.Username
	// TODO: There isn't a library that does this programatically, yet.
	if err := exec.Command("groupadd", "-g", u.Id, u.Username).Run(); err != nil {
		log.Fatalf("Failed to create group %s: %s", u.Username, err.Error())
	}
	if err := exec.Command("useradd", "-d", homeDir, "-u", u.Id, "-g", u.Username,
	    "-s", shell, u.Username).Run(); err != nil {
		log.Fatalf("Failed to create user %s: %s", u.Username, err.Error())
	}

	// Create systemd user slice
	sliceDir := "/etc/systemd/system/user-" + u.Id + ".slice.d"
	if err := os.Mkdir(sliceDir, 0755); err != nil {
		log.Fatal("Failed to create directory for user slice:", err)
	}
	of, err := os.OpenFile(sliceDir + "/override.conf", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Failed to create user slice override.conf:", err)
	}
	of.WriteString(u.SystemdLimits)
	of.Close()

	if u.PublicKey == "" { return }

	// Add key to authorized keys
	sshDir := homeDir + "/.ssh"
	if err := os.Mkdir(sshDir, 0700); err != nil {
		log.Fatal("Failed to create .ssh:", err)
	}
	id, err := strconv.Atoi(u.Id)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Chown(sshDir, id, id); err != nil {
		log.Fatal("Failed to chown .ssh:", err)
	}
	af, err := os.OpenFile(sshDir + "/authorized_keys",
	    os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal("Failed to open authorized_keys:", err)
	}
	if err := os.Chown(sshDir + "/authorized_keys", id, id); err != nil {
		log.Fatal("Failed to chown authorized_keys:", err)
	}
	af.WriteString(u.PublicKey)
	af.Close()
}
