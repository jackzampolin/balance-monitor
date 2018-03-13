package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

// TrackedBalances models multiple TrackedBalance
type TrackedBalances []TrackedBalance

// Addresses returns the list of addresses formatted for quering
func (tbs TrackedBalances) Addresses() string {
	out := ""

	for _, b := range tbs {
		out += fmt.Sprintf("%s|", b.Address)
	}

	return strings.TrimSuffix(out, "|")
}

// Address returns an individual tracked balance
func (tbs TrackedBalances) Address(addr string) TrackedBalance {
	for _, tb := range tbs {
		if tb.Address == addr {
			return tb
		}
	}
	return TrackedBalance{}
}

// TrackedBalance contains details necessary to track a balance
type TrackedBalance struct {
	Address         string `json:"address"`
	Service         string `json:"service"`
	Balance         int    `json:"balance"`
	NumTransactions int    `json:"numTransactions"`
}

// BlockchainInfoResponse is the response from an address query
type BlockchainInfoResponse map[string]BlockchainInfoBalance

// BlockchainInfoBalance is details from a query to blockchain.info/balance
type BlockchainInfoBalance struct {
	FinalBalance  int `json:"final_balance"`
	NTx           int `json:"n_tx"`
	TotalReceived int `json:"total_received"`
}

// FeeResponse models the response from the fees API
type FeeResponse struct {
	FastestFee  int `json:"fastestFee"`
	HalfHourFee int `json:"halfHourFee"`
	HourFee     int `json:"hourFee"`
}

// BalanceMonitor holds methods to query balances and write them to InfluxDB
type BalanceMonitor struct {
	TrackedBalances TrackedBalances
	BalanceAddr     string
	FeesAddr        string
	PollingInterval time.Duration
	InfluxClient    client.Client
	Port            string
}

// MakePoint takes all necessary data and returns an InfluxDB point
func MakePoint(fees *FeeResponse, tb TrackedBalance, bal BlockchainInfoBalance) *client.Point {
	measurement := "trackedBTCAddresses"
	tags := map[string]string{
		"address": tb.Address,
		"service": tb.Service,
	}
	fields := map[string]interface{}{
		"balance":        bal.FinalBalance,
		"totalReceived":  bal.TotalReceived,
		"alertThreshold": (tb.NumTransactions * fees.FastestFee),
	}
	pt, err := client.NewPoint(measurement, tags, fields, time.Now())
	if err != nil {
		panic(err)
	}
	return pt
}

// GetFees fetches the fee response data
func (bm *BalanceMonitor) GetFees() (*FeeResponse, error) {
	out := &FeeResponse{}
	res, err := http.Get(bm.FeesAddr)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MakePoints returns the points from all the monitored addresses
func (bm *BalanceMonitor) MakePoints() (client.BatchPoints, error) {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  "telegraf",
		Precision: "s",
	})

	if err != nil {
		return nil, err
	}

	fees, err := bm.GetFees()
	if err != nil {
		return nil, err
	}

	addrs, err := bm.GetAddressBalances()
	if err != nil {
		return nil, err
	}

	for addr, det := range *addrs {
		bp.AddPoint(MakePoint(fees, bm.TrackedBalances.Address(addr), det))
	}

	return bp, nil
}

// Monitor is the loop that continuously checks the address balance
func (bm *BalanceMonitor) Monitor() {
	for range time.Tick(bm.PollingInterval) {
		go func(bm *BalanceMonitor) {
			pts, err := bm.MakePoints()
			if err != nil {
				log.Println("Error making points", err)
			}
			err = bm.InfluxClient.Write(pts)
			if err != nil {
				log.Println("Error writing points", err)
			}
		}(bm)
	}
}

// GetAddressBalances returns the API response from the addresses query
func (bm *BalanceMonitor) GetAddressBalances() (*BlockchainInfoResponse, error) {
	out := &BlockchainInfoResponse{}
	res, err := http.Get(fmt.Sprintf(bm.BalanceAddr, bm.TrackedBalances.Addresses()))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NewBalanceMonitor returns a new BalanceMonitor
func NewBalanceMonitor(config *Config) *BalanceMonitor {
	dur, err := time.ParseDuration(config.PollingInterval)
	if err != nil {
		panic(err)
	}

	out := &BalanceMonitor{
		BalanceAddr:     config.BalanceAddress,
		FeesAddr:        config.FeesAddress,
		TrackedBalances: config.TrackedBalances,
		PollingInterval: dur,
		Port:            fmt.Sprintf(":%v", config.Port),
	}

	influxClient, err := client.NewHTTPClient(config.InfluxConfig.newHTTPConfig())
	if err != nil {
		panic(err)
	}

	out.InfluxClient = influxClient

	return out
}

// Config represents the configuration file used in this application
type Config struct {
	InfluxConfig    InfluxConfig    `json:"influxConfig"`
	TrackedBalances TrackedBalances `json:"trackedBalances"`
	BalanceAddress  string          `json:"balanceAddress"`
	FeesAddress     string          `json:"feesAddress"`
	PollingInterval string          `json:"pollingInterval"`
	Port            int             `json:"port"`
}

// InfluxConfig represents an InfluxDB connection
type InfluxConfig struct {
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (ic InfluxConfig) newHTTPConfig() client.HTTPConfig {
	return client.HTTPConfig{
		Addr:     ic.Address,
		Username: ic.Username,
		Password: ic.Password,
	}
}
