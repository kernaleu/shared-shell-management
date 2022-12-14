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
	"log"
	"os"
	"os/user"
	"os/exec"
	"errors"
)

func createUser(u systemUser) {
	exists, err := userExists(u.Username)
	if err != nil {
		log.Fatal(err)
	}
	if exists {
		return
	}

	createHomeDir(u)
	createUserSlice(u)
}

func userExists(username string) (bool, error) {
	userDoesNotExist := new(user.UnknownUserError)
	if _, err := user.Lookup(username); !errors.As(err, userDoesNotExist) {
		if err != nil {
			return false, err
		}
		// Skip pre existing user
		return true, nil
	}
	return false, nil
}

func createHomeDir(u systemUser) {
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

	if u.PublicKey == "" { return }

	// Add key to authorized keys
	sshDir := bdir + "/" + u.Username + "/.ssh"
	if err := os.Mkdir(sshDir, 0700); err != nil {
		log.Fatal("Failed to create directory for user slice:", err)
	}
	af, err := os.OpenFile(sshDir + "/authorized_keys",
	    os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal("Failed to open authorized_keys:", err)
	}
	af.WriteString(u.PublicKey)
	af.Close()

}

func createUserSlice(u systemUser) {
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
}
