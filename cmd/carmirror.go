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

var log = golog.Logger("kubo-car-mirror")

var background bool
var cid string
var addr string
var diff string

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
	// Args:  cobra.
	Run: func(cmd *cobra.Command, args []string) {
		var bgString string
		if background {
			bgString = "true"
		} else {
			bgString = "false"
		}

		var endpoint string
		if diff != "" {
			endpoint = fmt.Sprintf("/push/new?cid=%s&addr=%s&diff=%s&background=%s", cid, addr, diff, bgString)
		} else {
			endpoint = fmt.Sprintf("/push/new?cid=%s&addr=%s&background=%s", cid, addr, bgString)
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
		var bgString string
		if background {
			bgString = "true"
		} else {
			bgString = "false"
		}

		endpoint := fmt.Sprintf("/pull/new?cid=%s&addr=%s&background=%s", cid, addr, bgString)
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

	push.Flags().StringVarP(&cid, "cid", "c", "", "cid to push")
	push.Flags().StringVarP(&addr, "addr", "a", "", "remote address to push to")
	push.Flags().StringP("diff", "d", "", "diff against cid")
	push.Flags().BoolVarP(&background, "background", "b", false, "push in background")
	push.MarkFlagRequired("cid")
	push.MarkFlagRequired("addr")

	pull.Flags().StringVarP(&cid, "cid", "c", "", "cid to pull")
	pull.Flags().StringVarP(&addr, "addr", "a", "", "remote address to pull from")
	pull.Flags().BoolVarP(&background, "background", "b", false, "pull in background")
	pull.MarkFlagRequired("cid")
	pull.MarkFlagRequired("addr")

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
