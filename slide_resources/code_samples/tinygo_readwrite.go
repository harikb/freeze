
	bleChar.EnableNotifications( // HL
		func(buf []byte) { // HL
			println("data from ble device - length", len(buf))
			muteChan <- true
	})

	...
	...

	_, err = bleChar.WriteWithoutResponse( <your payload> ) // HL

	if err != nil { // this never happends for WriteWithoutResponse
		println(err)
	}


