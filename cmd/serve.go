// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

func ping(w http.ResponseWriter, r *http.Request) {
	message := r.URL.Path
	message = `{"status": "ok"}`
	w.Write([]byte(message))
}
func main() {

}

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "runs the balance-monitor",
	Run: func(cmd *cobra.Command, args []string) {
		bm := NewBalanceMonitor(config)
		go bm.Monitor()

		http.HandleFunc("/", ping)
		log.Printf("Bitcoin address monitor started on port %s...\n", bm.Port)
		if err := http.ListenAndServe(bm.Port, nil); err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
