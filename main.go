package main

import (
	"log"

	"github.com/varun425/MiniClubChaincode/erc20"
	"github.com/varun425/MiniClubChaincode/healthclub"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	miniclub, err := contractapi.NewChaincode(&healthclub.HealthClub{}, &erc20.SmartContract{})
	if err != nil {
		log.Panicf("Error creating miniclub chaincode: %v", err)
	}

	if err := miniclub.Start(); err != nil {
		log.Panicf("Error starting miniclub chaincode: %v", err)
	}
}
