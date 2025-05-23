// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package ssh

import (
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"

	"github.com/gliderlabs/ssh"
)

func listen(server *ssh.Server) {
	gracefulServer := graceful.NewServer("tcp", server.Addr, "SSH")
	gracefulServer.PerWriteTimeout = setting.SSH.PerWriteTimeout
	gracefulServer.PerWritePerKbTimeout = setting.SSH.PerWritePerKbTimeout

	err := gracefulServer.ListenAndServe(server.Serve, setting.SSH.UseProxyProtocol)
	if err != nil {
		select {
		case <-graceful.GetManager().IsShutdown():
			log.Critical("Failed to start SSH server: %v", err)
		default:
			log.Fatal("Failed to start SSH server: %v", err)
		}
	}
	log.Info("SSH Listener: %s Closed", server.Addr)
}

// builtinUnused informs our cleanup routine that we will not be using a ssh port
func builtinUnused() {
	graceful.GetManager().InformCleanup()
}
