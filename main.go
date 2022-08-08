package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"

	"github.com/noobdreamers/MiniClubChaincode/healthclub"
)

func main() {
	miniclub, err := contractapi.NewChaincode(&healthclub.HealthClub)
	if err != nil {
		log.Panicf("Error creating miniclub chaincode: %v", err)
	}

	if err := miniclub.Start(); err != nil {
		log.Panicf("Error starting tminiclub chaincode: %v", err)
	}
}
