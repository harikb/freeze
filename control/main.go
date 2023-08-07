// This example is intended to be used with the Adafruit Circuitplay Bluefruit board.
// It allows you to control the color of the built-in NeoPixel LEDS while they animate
// in a circular pattern.
package main

import (
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/bluetooth"
	"tinygo.org/x/drivers/ws2812"
)

var _g_RC_BLEAdapter = bluetooth.DefaultAdapter

// RO - Read-only variable, set only once
// RW - Frequently modified
// RC - Read and communicated with, but value not replaced

var (
	_g_RC_NEOPixels             machine.Pin = machine.NEOPIXELS
	_g_RC_LED                   machine.Pin = machine.LED
	_g_RC_ButtonA               machine.Pin = machine.BUTTONA
	_g_RC_ButtonB               machine.Pin = machine.BUTTONB
	_g_RC_WsHandle              ws2812.Device
	_g_RC_CommandCharacteristic bluetooth.Characteristic
)

var (
	_g_RO_ServiceUUID = [16]byte{0xa0, 0xb4, 0x00, 0x01, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3}
	_g_RO_CharUUID    = [16]byte{0xa0, 0xb4, 0x00, 0x02, 0x92, 0x6d, 0x4d, 0x61, 0x98, 0xdf, 0x8c, 0x5c, 0x62, 0xee, 0x53, 0xb3}
)

type LEDColor struct {
	r byte
	g byte
	b byte
}

var _g_RO_Green = LEDColor{r: 0x20, g: 0xff, b: 0x20}
var _g_RO_Red = LEDColor{r: 0xff, g: 0x20, b: 0x20}
var _g_RO_Black = LEDColor{r: 0x00, g: 0x00, b: 0x00}
var _g_RO_White = LEDColor{r: 0xff, g: 0xff, b: 0xff}

// TODO: use atomics to access *some of these* values.
var (
	_g_RW_StatusDisconnected bool = true
	_g_RW_StatusConnected    bool = false
	_g_RW_LEDStatusColor          = _g_RO_Red // start out with red
	_g_RW_PerLEDColors       [10]color.RGBA
)

var _g_RO_CMD_ConnectionEstablished = []byte{0x00, 0xFE, 0x01}
var _g_RO_CMD_MuteConfirmed = []byte{0x00, 0xFE, 0x02}
var _g_RO_CMD_MuteEnded = []byte{0x00, 0xFE, 0x03}
var _g_RO_CMD_EnterStandby = []byte{0x00, 0xFE, 0x04}

var _g_RO_CMD_MuteRequest = []byte{0x00, 0xBD, 0x03}
var _g_RO_CMD_Empty = []byte{0x00, 0x00, 0x00}

var colorChan chan LEDColor = make(chan LEDColor, 10)

func main() {
	// Start with a sleep. Without these, some startup error messages
	// are missed if the tingo console doesn't connect immediately after flashing
	time.Sleep(1 * time.Second)
	println("Starting TinyGo ")

	_g_RC_LED.Configure(machine.PinConfig{Mode: machine.PinOutput})
	_g_RC_NEOPixels.Configure(machine.PinConfig{Mode: machine.PinOutput})
	_g_RC_WsHandle = ws2812.New(_g_RC_NEOPixels)

	println("Configuring trigger buttons")
	_g_RC_ButtonA.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	_g_RC_ButtonB.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})

	_g_RC_BLEAdapter.SetConnectHandler(func(d bluetooth.Address, c bool) {
		_g_RW_StatusConnected = c
		if !_g_RW_StatusConnected && !_g_RW_StatusDisconnected {
			colorChan <- _g_RO_Black
			_g_RW_StatusDisconnected = true
		}

		if _g_RW_StatusConnected {
			_g_RW_StatusDisconnected = false
		}
	})

	must("enable BLE stack", _g_RC_BLEAdapter.Enable())
	adv := _g_RC_BLEAdapter.DefaultAdvertisement()
	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName: "Gltch Cntl",
	}))
	must("start adv", adv.Start())

	must("add service", _g_RC_BLEAdapter.AddService(&bluetooth.Service{
		UUID: bluetooth.NewUUID(_g_RO_ServiceUUID),
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &_g_RC_CommandCharacteristic,
				UUID:   bluetooth.NewUUID(_g_RO_CharUUID),
				Value:  _g_RO_CMD_Empty[:],
				Flags: bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicWritePermission |
					bluetooth.CharacteristicWriteWithoutResponsePermission | bluetooth.CharacteristicNotifyPermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					println("Received", len(value), value)
					if offset != 0 || len(value) != 3 {
						println("CORRUPTED payload - length", len(value))
						return
					}
					if !(value[0] == 0x00 && value[1] == 0xfe) {
						println("CORRUPTED payload - prefix", value[0], value[1], value[2])
						return
					}
					switch value[2] {
					case 0x01:
						colorChan <- _g_RO_White
					case 0x02:
						colorChan <- _g_RO_Green
					case 0x03:
						colorChan <- _g_RO_Red
					case 0x04:
						colorChan <- _g_RO_Black
					default:
						println("CORRUPTED payload - command", value[2])
						return
					}
				},
			},
		},
	}))

	blinking := true
	go ledManager()

	for {
		blinking = !blinking
		if _g_RC_ButtonA.Get() {
			if _g_RW_StatusConnected {
				println("Mute buttom pressed WHILE connected to Mac")
				n, err := _g_RC_CommandCharacteristic.Write(_g_RO_CMD_MuteRequest)
				if err != nil {
					println("Write to Mac failed!", err)
				}
				if n != len(_g_RO_CMD_MuteRequest) {
					println("Write to Mac failed! wrote only ", n)
				}
			} else {
				println("Mute buttom pressed, but NOT connected to Mac")
			}
		}

		// When *connected* the red LED will stay RED
		// When not connected the read LED will blink (as if to show BLE advertizement)
		_g_RC_LED.Set(_g_RW_StatusConnected || blinking)
		time.Sleep(100 * time.Millisecond)
	}
}

func ledManager() { // Expected to be run as a goroutine

	for clr := range colorChan {
		for i := range _g_RW_PerLEDColors {
			_g_RW_PerLEDColors[i] = color.RGBA{R: clr.r, G: clr.g, B: clr.b}
		}
		_g_RC_WsHandle.WriteColors(_g_RW_PerLEDColors[:])

	}
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}
