func freezer() {

	wallClock := time.NewTicker(1 * time.Second) // HL
	mutedTime := time.Now()

	for {
		select {
		case <-wallClock.C: // HL

			if muteState > 0 ........ {

			}

		case <-muteChan: // HL
			mutedTime = time.Now()
		....
