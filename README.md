# Coldlink

"unhotlinks" images optionally creating additional thumbnail versions.

As per example in `example` dir:


```
	cl := coldlink.Coldlink{StorageDir: ".", MaxOrigImageSizeInBytes: 10485760}
	result, err := cl.Get(
		"https://pixabay.com/static/uploads/photo/2016/09/30/11/54/owl-1705112_960_720.jpg",
		"owl",
		[]string{coldlink.OPT_ORIG, coldlink.OPT_SM, coldlink.OPT_XS},
	)
	if err != nil {
		fmt.Printf("Processing failed: %s", err.Error())
		os.Exit(1)
	}

	fmt.Printf("%+v\n", result)
```
