package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	golog "github.com/ipfs/go-log"
	"github.com/spf13/cobra"
)

var (
	defaultCmdAddr = "http://localhost:2502"
)

const pushBackgroundOptionName = "background"
const pullBackgroundOptionName = "background"

var log = golog.Logger("kubo-car-mirror")

// root command
var root = &cobra.Command{
	Use:   "carmirror",
	Short: "carmirror is a tool for efficiently diffing, deduplicating, packaging, and transmitting IPLD data from a source node to a sink node.",
	Long: `Requires a Kubo plugin. More details:
https://github.com/fission-codes/kubo-car-mirror`,
}

// push
var push *cobra.Command = &cobra.Command{
	Use:   "push",
	Short: "copy cid from local repo to remote addr",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		cid := args[0]
		addr := args[1]
		var background string
		if cmd.Flag(pushBackgroundOptionName).Value.String() == "true" {
			background = "true"
		} else {
			background = "false"
		}

		var endpoint string
		if len(args) == 3 {
			diff := args[2]
			endpoint = fmt.Sprintf("push/new?cid=%s&addr=%s&diff=%s&background=%s", cid, addr, diff, background)
		} else {
			endpoint = fmt.Sprintf("/push/new?cid=%s&addr=%s&background=%s", cid, addr, background)
		}

		res, err := doRemoteHTTPReq("POST", endpoint)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		// TODO: proper response handling
		fmt.Printf("pushed cid %s to:\n\t%s\n", cid, addr)
		fmt.Printf("response = %s\n", res)
	},
}

// pull
var pull = &cobra.Command{
	Use:   "pull",
	Short: "copy remote cid from remote addr to local repo",
	Run: func(cmd *cobra.Command, args []string) {
		cid := args[0]
		addr := args[1]
		var background string
		if cmd.Flag(pullBackgroundOptionName).Value.String() == "true" {
			background = "true"
		} else {
			background = "false"
		}

		endpoint := fmt.Sprintf("/pull/new?cid=%s&addr=%s&background=%s", cid, addr, background)
		_, err := doRemoteHTTPReq("POST", endpoint)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Printf("pulled cid %s from:\n\t%s\n", cid, addr)
	},
}

// ls
var ls = &cobra.Command{
	Use:   "ls",
	Short: "list all active transfers",
	Run: func(cmd *cobra.Command, args []string) {
		endpoint := "/ls"
		res, err := doRemoteHTTPReq("POST", endpoint)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, []byte(res), "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("sessions:\n%s\n", prettyJSON.Bytes())
	},
}

func init() {
	root.PersistentFlags().StringVar(&defaultCmdAddr, "commands-address", defaultCmdAddr, "address to issue requests that control local carmirror")
	push.Flags().BoolP(pushBackgroundOptionName, "b", false, "push in background")
	pull.Flags().BoolP(pullBackgroundOptionName, "b", false, "pull in background")
	root.AddCommand(push, pull, ls)
}

func main() {
	if err := root.Execute(); err != nil {
		fmt.Println(err)
	}
}

func doRemoteHTTPReq(method, endpoint string) (resMsg string, err error) {

	url := fmt.Sprintf("%s%s", defaultCmdAddr, endpoint)
	req, err := http.NewRequest(method, url, nil)
	log.Debugf("req = %v", req)
	if err != nil {
		return
	}

	res, err := http.DefaultClient.Do(req)
	log.Debugf("res = %v", res)
	if err != nil {
		return
	}
	defer res.Body.Close()

	log.Debugf("before reading all body, err=%v", err)
	resBytes, err := ioutil.ReadAll(res.Body)
	if resBytes == nil {
		return
	}

	resMsg = string(resBytes)
	return
}
