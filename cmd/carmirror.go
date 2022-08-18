package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	golog "github.com/ipfs/go-log"
	"github.com/spf13/cobra"
)

var (
	defaultCmdAddr = "http://localhost:2502"
)

var log = golog.Logger("car-mirror")

// root command
var root = &cobra.Command{
	Use:   "carmirror",
	Short: "carmirror is a tool for efficiently diffing, deduplicating, packaging, and transmitting IPLD data from a source node",
	Long: `Requires an IPFS plugin. More details:
https://github.com/fission-codes/go-car-mirror`,
}

// push
var push = &cobra.Command{
	Use:   "push",
	Short: "copy cid from local repo to remote addr",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		cid := args[0]
		addr := args[1]
		endpoint := fmt.Sprintf("/dag/push/new?cid=%s&addr=%s", cid, addr)

		_, err := doRemoteHTTPReq("POST", endpoint)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Printf("pushed cid %s to:\n\t%s\n", cid, addr)
	},
}

// pull
var pull = &cobra.Command{
	Use:   "pull",
	Short: "copy remote cid from peer to local repo",
	Run: func(cmd *cobra.Command, args []string) {
		cid := args[0]
		addr := args[1]

		endpoint := fmt.Sprintf("/dag/pull/new?cid=%s&addr=%s", cid, addr)
		_, err := doRemoteHTTPReq("POST", endpoint)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		// fmt.Println(res)
		fmt.Printf("pulled cid %s from:\n\t%s\n", cid, addr)
	},
}

func init() {
	root.PersistentFlags().StringVar(&defaultCmdAddr, "command-address", defaultCmdAddr, "address to issue requests that control local carmirror")
	root.AddCommand(push, pull)
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

	resBytes, err := ioutil.ReadAll(res.Body)
	if resBytes == nil {
		return
	}

	resMsg = string(resBytes)
	return
}
