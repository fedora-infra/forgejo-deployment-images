// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package asymkey

import (
	"context"
	"fmt"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/modules/log"
	"forgejo.org/modules/process"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"

	"golang.org/x/crypto/ssh"
	"xorm.io/builder"
)

// ___________.__                                         .__        __
// \_   _____/|__| ____    ____   ________________________|__| _____/  |_
//  |    __)  |  |/    \  / ___\_/ __ \_  __ \____ \_  __ \  |/    \   __\
//  |     \   |  |   |  \/ /_/  >  ___/|  | \/  |_> >  | \/  |   |  \  |
//  \___  /   |__|___|  /\___  / \___  >__|  |   __/|__|  |__|___|  /__|
//      \/            \//_____/      \/      |__|                 \/
//
// This file contains functions for fingerprinting SSH keys
//
// The database is used in checkKeyFingerprint however most of these functions probably belong in a module

// checkKeyFingerprint only checks if key fingerprint has been used as public key,
// it is OK to use same key as deploy key for multiple repositories/users.
func checkKeyFingerprint(ctx context.Context, fingerprint string) error {
	has, err := db.Exist[PublicKey](ctx, builder.Eq{"fingerprint": fingerprint})
	if err != nil {
		return err
	} else if has {
		return ErrKeyAlreadyExist{0, fingerprint, ""}
	}
	return nil
}

func calcFingerprintSSHKeygen(publicKeyContent string) (string, error) {
	// Calculate fingerprint.
	tmpPath, err := writeTmpKeyFile(publicKeyContent)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := util.Remove(tmpPath); err != nil {
			log.Warn("Unable to remove temporary key file: %s: Error: %v", tmpPath, err)
		}
	}()
	stdout, stderr, err := process.GetManager().Exec("AddPublicKey", "ssh-keygen", "-lf", tmpPath)
	if err != nil {
		if strings.Contains(stderr, "is not a public key file") {
			return "", ErrKeyUnableVerify{stderr}
		}
		return "", util.NewInvalidArgumentErrorf("'ssh-keygen -lf %s' failed with error '%s': %s", tmpPath, err, stderr)
	} else if len(stdout) < 2 {
		return "", util.NewInvalidArgumentErrorf("not enough output for calculating fingerprint: %s", stdout)
	}
	return strings.Split(stdout, " ")[1], nil
}

func calcFingerprintNative(publicKeyContent string) (string, error) {
	// Calculate fingerprint.
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKeyContent))
	if err != nil {
		return "", err
	}
	return ssh.FingerprintSHA256(pk), nil
}

// CalcFingerprint calculate public key's fingerprint
func CalcFingerprint(publicKeyContent string) (string, error) {
	// Call the method based on configuration
	useNative := setting.SSH.KeygenPath == ""
	calcFn := util.Iif(useNative, calcFingerprintNative, calcFingerprintSSHKeygen)
	fp, err := calcFn(publicKeyContent)
	if err != nil {
		if IsErrKeyUnableVerify(err) {
			return "", err
		}
		return "", fmt.Errorf("CalcFingerprint(%s): %w", util.Iif(useNative, "native", "ssh-keygen"), err)
	}
	return fp, nil
}
