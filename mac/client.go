package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

var (
	adapter = bluetooth.DefaultAdapter

	serviceUUID = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x00, 0x01, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
	charUUID    = bluetooth.NewUUID([16]byte{0xa0, 0xb4, 0x00, 0x02, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3})
)

var _g_RO_CMD_ConnectionEstablished = []byte{0x00, 0xFE, 0x01}
var _g_RO_CMD_MuteConfirmed = []byte{0x00, 0xFE, 0x02}
var _g_RO_CMD_MuteEnded = []byte{0x00, 0xFE, 0x03}
var _g_RO_CMD_EnterStandby = []byte{0x00, 0xFE, 0x04}

var _g_RO_CMD_MuteRequest = []byte{0x00, 0xBD, 0x03}

var foundDevices sync.Map

var muteChan chan bool = make(chan bool)

var _g_RC_BLEService bluetooth.Service
var _g_RC_BLEChar bluetooth.DeviceCharacteristic

func main() {
	println("enabling")

	// Enable BLE interface.
	must("enable BLE stack", adapter.Enable())

	ch := make(chan bluetooth.ScanResult, 10)

	// Start scanning.
	println("scanning...")
	err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		_, loaded := foundDevices.LoadOrStore(result.Address.String(), result.LocalName())
		if !loaded {
			println("found device:", result.Address.String(), result.RSSI, result.LocalName())

			if result.LocalName() == "Gltch Cntl" {
				adapter.StopScan()
				ch <- result
			}
		}
	})

	var device *bluetooth.Device
	select {
	case result := <-ch:
		device, err = adapter.Connect(result.Address, bluetooth.ConnectionParams{})
		if err != nil {
			println(err.Error())
			return
		}

		println("connected to ", result.Address.String())
	}

	// get services
	println("discovering services/characteristics")
	srvcs, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})

	must("discover services", err)

	if len(srvcs) == 0 {
		panic("could not find heart rate service")
	}

	_g_RC_BLEService := srvcs[0]

	println("found service", _g_RC_BLEService.UUID().String())

	chars, err := _g_RC_BLEService.DiscoverCharacteristics([]bluetooth.UUID{charUUID})
	if err != nil {
		println(err)
	}
	println("Discovered", len(chars), " characteristics")

	if len(chars) == 0 {
		panic("could not find heart rate characteristic")
	}

	_g_RC_BLEChar = chars[0]
	println("found characteristic", _g_RC_BLEChar.UUID().String())

	_g_RC_BLEChar.EnableNotifications(func(buf []byte) {
		println("data from ble device - length", len(buf))
		muteChan <- true
	})

	go freezer()

	_, err = _g_RC_BLEChar.WriteWithoutResponse(_g_RO_CMD_ConnectionEstablished)
	if err != nil { // this never happends for WriteWithoutResponse
		println(err)
	}

	select {}
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}

func mute() error {

	_, err := exec.Command("osascript", "-e", "set volume input volume 0").CombinedOutput()
	if err != nil {
		println(err)
		return err
	}

	out, err := exec.Command("osascript", "-e", "input volume of (get volume settings)").CombinedOutput()
	if err != nil {
		println(err)
		return err
	}
	vol, err := strconv.Atoi(string(out))
	if err != nil {
		println(err)
		return err
	}
	if vol != 0 {
		err = fmt.Errorf("unable to set the input volumn to 0")
		println(err)
		return err
	}
	return nil
}

func unmute() error {

	_, err := exec.Command("osascript", "-e", "set volume input volume 80").CombinedOutput()
	if err != nil {
		println(err)
		return err
	}

	out, err := exec.Command("osascript", "-e", "input volume of (get volume settings)").CombinedOutput()
	if err != nil {
		println(err)
		return err
	}
	vol, err := strconv.Atoi(string(out))
	if err != nil {
		println(err)
		return err
	}
	if vol != 80 {
		err = fmt.Errorf("unable to restore input volumn to 80")
		println(err)
		return err
	}
	return nil
}

func freezer() {

	wallClock := time.NewTicker(1 * time.Second)
	mutedTime := time.Now()
	muteState := 0

	time.Sleep(5 * time.Second)
	_, err := _g_RC_BLEChar.WriteWithoutResponse(_g_RO_CMD_EnterStandby)
	if err != nil { // this never happends for WriteWithoutResponse
		println(err)
	}

	for {

		select {

		case <-wallClock.C:

			if muteState > 0 && time.Since(mutedTime) > (time.Second*15) {

				muteState = 0
				_, err := _g_RC_BLEChar.WriteWithoutResponse(_g_RO_CMD_EnterStandby)
				if err != nil { // this never happends for WriteWithoutResponse
					println(err)
				}
			}

			if muteState > 1 && time.Since(mutedTime) > (time.Second*10) {

				muteState = 1 // try only once - avoid infinite loops

				// TODO: DO UNMUTE
				err = unmute()
				if err != nil {
					_, err := _g_RC_BLEChar.WriteWithoutResponse(_g_RO_CMD_MuteEnded)
					if err != nil { // this never happends for WriteWithoutResponse
						println(err)
					}
				}
			}

		case <-muteChan:
			mutedTime = time.Now()

			// TODO: DO MUTE
			err = mute()
			if err != nil {

				muteState = 2
				_, err := _g_RC_BLEChar.WriteWithoutResponse(_g_RO_CMD_MuteConfirmed)
				if err != nil { // this never happends for WriteWithoutResponse
					println(err)
				}
			}
		}
	}
}
