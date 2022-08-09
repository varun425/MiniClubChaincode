package healthclub

import (
	"encoding/json"
	"fmt"
	"log"

	erc20 "github.com/VisheshSolu/MiniClubChaincode/erc20"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type HealthClub struct {
	contractapi.Contract
	erc20.SmartContract
}

type User struct {
	Memberships []string `json:"memberships"`
	Name        string   `json:"name"`
	Email       string   `json:"email`
}

type Level struct {
	EntryPrizeTokens uint `json:"entryprizetokens"`
	Months           uint `json:"months"`
}

const userPrefix = "User-"

func (h *HealthClub) RegisterUser(ctx contractapi.TransactionContextInterface, name string, email string) (string, error) {

	userid, err := ctx.GetClientIdentity().GetID()

	if err != nil {
		return "", fmt.Errorf("failed to get userID: %v", err)
	}

	// check user is already registered or not
	newUserId := userPrefix + userid

	user, err := ctx.GetStub().GetState(newUserId)
	if err != nil {
		return "", fmt.Errorf("error:%v", err)
	}

	if user != nil {
		return "", fmt.Errorf("User %v already registered", newUserId)
	}

	// create new user
	userdetails := User{
		Memberships: []string{},
		Name:        name,
		Email:       email,
	}

	userdetailsbytes, _ := json.Marshal(userdetails)
	err = ctx.GetStub().PutState(newUserId, userdetailsbytes)

	if err != nil {
		return "", fmt.Errorf("error:%v", err)
	}

	// send bonus token to user account
	err = h.Mint(ctx, 100)
	if err != nil {
		return "", fmt.Errorf("error:%v", err)
	}

	log.Printf("%v registered successfully", newUserId)
	return "User registered successfully", nil
}

func (h *HealthClub) SetMembershipLevelToken(ctx contractapi.TransactionContextInterface, level string, months uint, entryPrizeTokens uint) error {

	res, roleBool, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil {
		return fmt.Errorf("error:%v", err)
	}

	if !roleBool {
		return fmt.Errorf("attribute-value is not set")
	}

	if res != "admin" {
		return fmt.Errorf("current role is :%v ,but requires admin role", res)
	}

	if level == "gold" || level == "diamond" || level == "platinum" {

		temp := Level{
			EntryPrizeTokens: entryPrizeTokens,
			Months:           months,
		}

		resInBytes, err := json.Marshal(temp)
		if err != nil {
			return fmt.Errorf("error:%v", err)
		}

		err = ctx.GetStub().PutState(level, resInBytes)
		if err != nil {
			return fmt.Errorf("error:%v", err)
		}

		log.Printf("The %v level is set at %v tokens for %v months ", level, entryPrizeTokens, months)

	} else {
		return fmt.Errorf("only gold, diamond, and platinum levels are acceptable")
	}

	return nil

}
