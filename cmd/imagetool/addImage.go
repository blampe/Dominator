package main

import (
	"errors"
	"fmt"
	"net/rpc"
	"os"
)

func addImageSubcommand(client *rpc.Client, args []string) {
	err := addImage(client, args[0], args[1], args[2])
	if err != nil {
		fmt.Printf("Error adding image: \"%s\"\t%s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addImage(client *rpc.Client,
	name, imageFilename, filterFilename string) error {
	imageFile, err := os.Open(imageFilename)
	if err != nil {
		return err
	}
	defer imageFile.Close()
	filterFile, err := os.Open(filterFilename)
	if err != nil {
		return err
	}
	defer filterFile.Close()
	imageExists, err := checkImage(client, name)
	if err != nil {
		return err
	}
	if imageExists {
		return errors.New("image exists")
	}
	return nil
}