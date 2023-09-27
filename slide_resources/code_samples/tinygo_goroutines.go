	go ledManager() // HL
	...
}

// ledManager goroutine reads LED colors from a channel
func ledManager() {  // HL

	for clr := range colorChan { // HL
		...
		x.WriteColors(...)
	}
}

