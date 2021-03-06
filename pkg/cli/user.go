// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Author: Marc Berhault (marc@cockroachlabs.com)

package cli

import (
	"bufio"
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/cockroachdb/cockroach/pkg/security"
)

var password string

// A getUserCmd command displays the config for the specified username.
var getUserCmd = &cobra.Command{
	Use:   "get [options] <username>",
	Short: "fetches and displays a user",
	Long: `
Fetches and displays the user for <username>.
`,
	SilenceUsage: true,
	RunE:         maybeDecorateGRPCError(runGetUser),
}

func runGetUser(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return usageAndError(cmd)
	}
	conn, err := makeSQLClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	return runQueryAndFormatResults(conn, os.Stdout,
		makeQuery(`SELECT * FROM system.users WHERE username=$1`, args[0]), cliCtx.prettyFmt)
}

// A lsUsersCmd command displays a list of users.
var lsUsersCmd = &cobra.Command{
	Use:   "ls [options]",
	Short: "list all users",
	Long: `
List all users.
`,
	SilenceUsage: true,
	RunE:         maybeDecorateGRPCError(runLsUsers),
}

func runLsUsers(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return usageAndError(cmd)
	}
	conn, err := makeSQLClient()
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	return runQueryAndFormatResults(conn, os.Stdout,
		makeQuery(`SELECT username FROM system.users`), cliCtx.prettyFmt)
}

// A rmUserCmd command removes the user for the specified username.
var rmUserCmd = &cobra.Command{
	Use:   "rm [options] <username>",
	Short: "remove a user",
	Long: `
Remove an existing user by username.
`,
	SilenceUsage: true,
	RunE:         maybeDecorateGRPCError(runRmUser),
}

func runRmUser(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return usageAndError(cmd)
	}
	conn, err := makeSQLClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	return runQueryAndFormatResults(conn, os.Stdout,
		makeQuery(`DELETE FROM system.users WHERE username=$1`, args[0]), cliCtx.prettyFmt)
}

// A setUserCmd command creates a new or updates an existing user.
var setUserCmd = &cobra.Command{
	Use:   "set [options] <username>",
	Short: "create or update a user",
	Long: `
Create or update a user for the specified username, prompting
for the password.
`,
	SilenceUsage: true,
	RunE:         maybeDecorateGRPCError(runSetUser),
}

// runSetUser prompts for a password, then inserts the user and hash
// into the system.users table.
// TODO(marc): once we have more fields in the user, we will need
// to allow changing just some of them (eg: change email, but leave password).
func runSetUser(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return usageAndError(cmd)
	}
	var err error
	var hashed []byte
	switch password {
	case "":
		hashed, err = security.PromptForPasswordAndHash()
		if err != nil {
			return err
		}
	case "-":
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			if b := scanner.Bytes(); len(b) > 0 {
				hashed, err = security.HashPassword(b)
				if err != nil {
					return err
				}
				if scanner.Scan() {
					return errors.New("multiline passwords are not permitted")
				}
				if err := scanner.Err(); err != nil {
					return err
				}

				break // Success.
			}
		} else {
			if err := scanner.Err(); err != nil {
				return err
			}
		}

		panic("empty passwords are not permitted")
	default:
		hashed, err = security.HashPassword([]byte(password))
		if err != nil {
			return err
		}
	}
	conn, err := makeSQLClient()
	if err != nil {
		return err
	}
	defer conn.Close()
	// TODO(marc): switch to UPSERT.
	return runQueryAndFormatResults(conn, os.Stdout,
		makeQuery(`INSERT INTO system.users VALUES ($1, $2)`, args[0], hashed), cliCtx.prettyFmt)
}

var userCmds = []*cobra.Command{
	getUserCmd,
	lsUsersCmd,
	rmUserCmd,
	setUserCmd,
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "get, set, list and remove users",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Usage()
	},
}

func init() {
	userCmd.AddCommand(userCmds...)
}
