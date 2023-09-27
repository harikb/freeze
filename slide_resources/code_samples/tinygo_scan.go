	println("scanning...")

	err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {

		if result.LocalName() == "Freeze Cntrl" { // HL

			adapter.StopScan()
			ch <- result
		}
	})

