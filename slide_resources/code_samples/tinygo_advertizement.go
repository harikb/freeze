	bleAdapter.Enable()
	adv := bleAdapter.DefaultAdvertisement()
	err := adv.Configure(bluetooth.AdvertisementOptions{

		LocalName: "Freeze Cntrl", // HL

	})
	...
	adv.Start()
