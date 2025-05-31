// Copyright © 2016 John Mylchreest <jmylchreest@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/ipv4"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the IGMPQD daemon",
	Long: `
`,
	Run: func(cmd *cobra.Command, args []string) {
		debug("IGMPQD Started.")
		debug(fmt.Sprintf("%15s: %s", "version", GitDescribe))
		debug(fmt.Sprintf("%15s: %s", "commit", GitCommit))
		debug(fmt.Sprintf("%15s: %s", "grpAddress", viper.GetString("grpAddress")))
		debug(fmt.Sprintf("%15s: %s", "dstAddress", viper.GetString("dstAddress")))
		debug(fmt.Sprintf("%15s: %s", "interface", viper.GetString("interface")))
		debug(fmt.Sprintf("%15s: %d", "maxResponseTime", viper.GetInt("maxResponseTime")))
		debug(fmt.Sprintf("%15s: %d", "interval", viper.GetInt("interval")))
		debug(fmt.Sprintf("%15s: %d", "ttl", viper.GetInt("ttl")))

		tickC := time.NewTicker(time.Second * time.Duration(viper.GetInt("interval"))).C
		signalC := make(chan os.Signal, 1)
		signal.Notify(signalC, os.Interrupt)
		go func() {
			sendPacket()
			for range tickC { // More idiomatic way to range when value is not used
				sendPacket()
			}
		}()
		<-signalC
	},
}

func sendPacket() {
	var payload []byte

	grpAddress := net.ParseIP(viper.GetString("grpAddress"))
	dstAddress := net.ParseIP(viper.GetString("dstAddress"))

	// IGMP https://tools.ietf.org/html/rfc2236#section-2
	payload = make([]byte, 8) // Simplified when length and capacity are the same
	payload[0] = uint8(0x11)
	payload[1] = uint8(viper.GetInt("maxResponseTime"))
	payload[4] = grpAddress.To4()[0]
	payload[5] = grpAddress.To4()[1]
	payload[6] = grpAddress.To4()[2]
	payload[7] = grpAddress.To4()[3]
	binary.BigEndian.PutUint16(payload[2:], genChecksum(payload, 0))

	debug("opening socket.")
	// inspired by https://godoc.org/golang.org/x/net/ipv4#example-RawConn--AdvertisingOSPFHello
	c, err := net.ListenPacket("ip4:2", "0.0.0.0") // IGMP
	if err != nil {
		log.Fatal("Error occured. ", err)
	}
	defer debug("closing socket.")
	defer c.Close()

	r, err := ipv4.NewRawConn(c)
	if err != nil {
		log.Fatal("Error occured. ", err)
	}

	iph := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TOS:      0xc0, // DSCP CS6
		TotalLen: ipv4.HeaderLen + len(payload),
		TTL:      viper.GetInt("ttl"),
		Protocol: 2,
		Dst:      dstAddress.To4(),
	}

	var cm *ipv4.ControlMessage

	interfaceName := viper.GetString("interface")
	if interfaceName != "" {
		netif, err := net.InterfaceByName(interfaceName)
		if err != nil {
			log.Fatal(err)
		}
		switch runtime.GOOS {
		case "darwin", "linux":
			cm = &ipv4.ControlMessage{IfIndex: netif.Index}
		default:
			if err := r.SetMulticastInterface(netif); err != nil {
				log.Fatal(err)
			}
		}
	}

	if err := r.WriteTo(iph, payload, cm); err != nil {
		log.Fatal(err)
	}

}

// SetupRunCommand configures the flags and Viper bindings for the given command.
// This is used by init() and can be called by tests.
func SetupRunCommand(cmdToSetup *cobra.Command) {
	cmdToSetup.PersistentFlags().Bool("debug", false, "Enable debug messages to stderr.")
	cmdToSetup.PersistentFlags().StringP("grpAddress", "g", "0.0.0.0", "Specified IP address to use as the Group Address. Used to query for specific group members.")
	cmdToSetup.PersistentFlags().StringP("dstAddress", "d", "224.0.0.1", "Specified IP address to send the IGMP Query to.")
	cmdToSetup.PersistentFlags().StringP("interface", "I", "", "Specified network interface to send the IGMP Query.")
	cmdToSetup.PersistentFlags().IntP("interval", "i", 30, "The time in seconds to delay between sending IGMP Query messages.")
	cmdToSetup.PersistentFlags().IntP("ttl", "t", 1, "The TTL of the IGMP Query.")
	cmdToSetup.PersistentFlags().IntP("maxResponseTime", "m", 100, "Specifies the maximum allowed time before sending a responding report in units of 1/10 second.")

	viper.BindPFlag("debug", cmdToSetup.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("grpAddress", cmdToSetup.PersistentFlags().Lookup("grpAddress"))
	viper.BindPFlag("dstAddress", cmdToSetup.PersistentFlags().Lookup("dstAddress"))
	viper.BindPFlag("interface", cmdToSetup.PersistentFlags().Lookup("interface"))
	viper.BindPFlag("interval", cmdToSetup.PersistentFlags().Lookup("interval"))
	viper.BindPFlag("ttl", cmdToSetup.PersistentFlags().Lookup("ttl"))
	viper.BindPFlag("maxResponseTime", cmdToSetup.PersistentFlags().Lookup("maxResponseTime"))
}

func init() {
	RootCmd.AddCommand(runCmd)
	SetupRunCommand(runCmd) // Call the new setup function
}

func debug(message string) {
	if viper.GetBool("debug") {
		log.Println(message)
	}
}

// Calculate the TCP/IP checksum defined in rfc1071.
func genChecksum(data []byte, csum uint32) uint16 {
	length := len(data) - 1
	for i := 0; i < length; i += 2 {
		csum += uint32(data[i]) << 8
		csum += uint32(data[i+1])
	}
	if len(data)%2 == 1 {
		csum += uint32(data[length]) << 8
	}
	for csum > 0xffff {
		csum = (csum & 0xffff) + (csum >> 16)
	}
	return ^uint16((csum >> 16) + csum)
}
