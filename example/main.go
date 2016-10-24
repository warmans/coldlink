package main

import (
	"fmt"
	"os"

	"github.com/warmans/coldlink"
)

func main() {

	cl := coldlink.Coldlink{StorageDir: ".", MaxOrigImageSizeInBytes: 10485760}
	result, err := cl.Get(
		"https://pixabay.com/static/uploads/photo/2016/09/30/11/54/owl-1705112_960_720.jpg",
		"owl",
		[]*coldlink.TargetSpec{
			{Name: "orig", Op: coldlink.OpOriginal},
			{Name: "sm", Op: coldlink.OpThumb, Width: 150, Height: 150},
			{Name: "xs", Op: coldlink.OpThumb, Width: 50, Height: 50},
		},
	)
	if err != nil {
		fmt.Printf("Processing failed: %s", err.Error())
		os.Exit(1)
	}

	fmt.Printf("%+v\n", result)
}
