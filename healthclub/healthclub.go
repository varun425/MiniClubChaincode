package healthclub

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

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
	Email       string   `json:"email"`
}

type Level struct {
	EntryPrizeTokens int `json:"entryprizetokens"`
	Months           int `json:"months"`
}

type Membership struct {
	Level          string `json:"level"`
	TokenDeposited int
	IsCompleted    bool
	IsCancelled    bool
	IsUpdated      bool
	StartDate      string
	EndDate        string
	RefundAmount   int
	UserID         string
}

const (
	userPrefix       = "User-"
	membershipPrefix = "Membership-"
	goldlevel        = "Gold"
	platinumlevel    = "Platinum"
	diamondlevel     = "Diamond"
)

func (h *HealthClub) InitializeContract(ctx contractapi.TransactionContextInterface) error {

	ownerid, err := ctx.GetStub().GetState("owner")
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	if ownerid != nil {
		return fmt.Errorf("already initialzed")
	}

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

	adminID, _ := ctx.GetClientIdentity().GetID()
	err = ctx.GetStub().PutState("owner", []byte(adminID))

	if err != nil {
		return fmt.Errorf("error:%v", err)
	}

	isinitialize, err := h.Initialize(ctx, "MiniFitnessHealthClub", "MFHC", "18")
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	if !isinitialize {
		return fmt.Errorf("not able to initialize contract")
	}

	err = setMembershipLevelToken(ctx, "Gold", 1, 1000)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	err = setMembershipLevelToken(ctx, "Platinum", 6, 5000)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	err = setMembershipLevelToken(ctx, "Diamond", 12, 8000)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}

	err = ctx.GetStub().PutState("TotalMemberships", []byte(strconv.Itoa(0)))
	if err != nil {
		return fmt.Errorf("error:%v", err)
	}

	return nil
}

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

func setMembershipLevelToken(ctx contractapi.TransactionContextInterface, level string, months int, entryPrizeTokens int) error {

	if level == goldlevel || level == diamondlevel || level == platinumlevel {

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
		return fmt.Errorf("only Gold, Diamond, and Platinum levels are acceptable")
	}

	return nil
}

func (h *HealthClub) GetNewMemberShip(ctx contractapi.TransactionContextInterface, level string) (string, error) {

	userid, err := ctx.GetClientIdentity().GetID()

	if err != nil {
		return "", fmt.Errorf("failed to get userID: %v", err)
	}

	userId := userPrefix + userid

	user, err := ctx.GetStub().GetState(userId)
	if err != nil {
		return "", fmt.Errorf("error:%v", err)
	}

	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	if level == goldlevel || level == diamondlevel || level == platinumlevel {

		levelbytes, err := ctx.GetStub().GetState(level)

		if err != nil {
			return "", fmt.Errorf("err:%s", err.Error())
		}

		if levelbytes == nil {
			return "", fmt.Errorf("%s level does not exist", level)
		}

		levelptr := new(Level)
		_ = json.Unmarshal(levelbytes, &levelptr)

		currentTime := time.Now()
		endTime := currentTime.AddDate(0, levelptr.Months, 0)

		membership := Membership{
			Level:          level,
			TokenDeposited: levelptr.EntryPrizeTokens,
			IsCompleted:    false,
			IsCancelled:    false,
			IsUpdated:      false,
			StartDate:      currentTime.Format("01-02-2006"),
			EndDate:        endTime.Format("01-02-2006"),
			RefundAmount:   0,
			UserID:         userId,
		}

		// create new membership
		membershipAsBytes, _ := json.Marshal(membership)

		TotalMembershipsbytes, err := ctx.GetStub().GetState("TotalMemberships")

		if err != nil {
			return "", fmt.Errorf("err: %v", err)
		}

		TotalMemberships, _ := strconv.Atoi(string(TotalMembershipsbytes))

		membershipID := membershipPrefix + strconv.Itoa(TotalMemberships+1)
		ctx.GetStub().PutState(membershipID, membershipAsBytes)

		updatedTotalMemberships := TotalMemberships + 1

		err = ctx.GetStub().PutState("TotalMemberships", []byte(strconv.Itoa(updatedTotalMemberships)))

		if err != nil {
			return "", fmt.Errorf("err: %v", err)
		}

		// update user memberships
		userptr := new(User)
		_ = json.Unmarshal(user, &userptr)

		log.Printf("user details: %v", (userptr))

		if len(userptr.Memberships) != 0 {
			currentmembershipId := userptr.Memberships[len(userptr.Memberships)-1]
			membershipdetailsBytes, err := ctx.GetStub().GetState(currentmembershipId)
			if err != nil {
				return "", fmt.Errorf("error: %v", err)
			}

			membershipdetails := new(Membership)
			_ = json.Unmarshal(membershipdetailsBytes, &membershipdetails)
			now := time.Now()
			membershipendDate, _ := time.Parse("01-02-2006", membershipdetails.EndDate)
			compareTime := now.After(membershipendDate)

			if compareTime {
				membershipdetails.IsCompleted = true
				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}
			} else if membershipdetails.IsCancelled {
				// do nothing
			} else {
				return "", fmt.Errorf("memebership not ended, Please wait for current membership to end")
			}
		}

		userptr.Memberships = append(userptr.Memberships, membershipID)

		userdetailsbytes, _ := json.Marshal(userptr)
		err = ctx.GetStub().PutState(userId, userdetailsbytes)

		if err != nil {
			return "", fmt.Errorf("error:%v", err)
		}

		// Update the state of the smart contract by adding the allowanceKey and value

		var index string = "level~UserID"
		//userdetailsbytes, _ := json.Marshal(userptr)
		// ctx.GetStub().PutState(level, userdetailsbytes)
		userLevelIndexKey, err := ctx.GetStub().CreateCompositeKey(index, []string{level, userId})
		if err != nil {
			return "", fmt.Errorf("failed to create the composite key for prefix %s: %v", index, err)
		}
		// value := []byte{0x00}
		// ctx.GetStub().PutState(userLevelIndexKey, value)
		// fmt.Println("Added", userId)
		err = ctx.GetStub().PutState(userLevelIndexKey, userdetailsbytes)
		if err != nil {
			return "", fmt.Errorf("error %v", err)
		}

		log.Printf("user memberships updated successfully")

		adminidbytes, err := ctx.GetStub().GetState("owner")

		if err != nil {
			return "", fmt.Errorf("err: %v", err)
		}

		if adminidbytes == nil {
			return "", fmt.Errorf("AdminID not set")
		}

		adminID := string(adminidbytes)

		err = h.Transfer(ctx, adminID, levelptr.EntryPrizeTokens)

		if err != nil {
			return "", fmt.Errorf("err: %v", err)
		}

		return "Successfully get new Membership", nil
		// transfer tokens from user to admin

	} else {
		return "", fmt.Errorf("invalid level")
	}
}

func (h *HealthClub) GetAllMembershipByLevel(ctx contractapi.TransactionContextInterface, level string) ([]string, error) {
	var index string = "level~UserID"
	levelUsers := []string{}

	//return bytes memory address of first state
	levelIterator, err := ctx.GetStub().GetStateByPartialCompositeKey(index, []string{level})
	if err != nil {
		return nil, fmt.Errorf("error:%v", err.Error())
	}

	defer levelIterator.Close()

	// Iterate through result set and for each level found
	var i int
	for i = 0; levelIterator.HasNext(); i++ {
		responseRange, err := levelIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("error:%v", err.Error())
		}
		// get the level and userID from defined index composite key
		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(responseRange.Key)
		if err != nil {
			return nil, fmt.Errorf("error:%v", err.Error())
		}
		returnedLevelUserID := compositeKeyParts[1]
		levelUsers = append(levelUsers, returnedLevelUserID)
	}
	return levelUsers, nil
}

func (h *HealthClub) CancelMembership(ctx contractapi.TransactionContextInterface) (string, error) {

	userid, err := ctx.GetClientIdentity().GetID()

	if err != nil {
		return "", fmt.Errorf("failed to get userID: %v", err)
	}

	userId := userPrefix + userid

	user, err := ctx.GetStub().GetState(userId)
	if err != nil {
		return "", fmt.Errorf("error:%v", err)
	}

	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	userptr := new(User)
	_ = json.Unmarshal(user, &userptr)

	log.Printf("user details: %v", (userptr))

	if len(userptr.Memberships) == 0 {
		return "", fmt.Errorf(("no membership found"))
	} else {
		currentmembershipId := userptr.Memberships[len(userptr.Memberships)-1]
		membershipdetailsBytes, err := ctx.GetStub().GetState(currentmembershipId)

		log.Printf("mid", currentmembershipId)

		membershipdetails := new(Membership)
		_ = json.Unmarshal(membershipdetailsBytes, &membershipdetails)

		log.Printf("mdetails", membershipdetails)

		if membershipdetails.Level == goldlevel {

			refundamount := 0
			log.Printf("refund amount will be %v", refundamount)

			if err != nil {
				return "", fmt.Errorf("error: %v", err)
			}

			membershipdetails.RefundAmount = refundamount
			membershipdetails.IsCancelled = true
			membershipdetails.IsCompleted = true

			updatedmembership, _ := json.Marshal(membershipdetails)
			err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
			if err != nil {
				return "", fmt.Errorf("error:%v", err)
			}

			return "Successfully Cancel gold Membership", nil

		} else if membershipdetails.Level == platinumlevel {
			membershipstartDate, _ := time.Parse("01-02-2006", membershipdetails.StartDate)
			now := time.Now()

			onemonthplusbuffer := membershipstartDate.AddDate(0, 1, 7)
			twomonthplusbuffer := membershipstartDate.AddDate(0, 2, 7)
			threemonthplusbuffer := membershipstartDate.AddDate(0, 3, 7)
			fourmonthplusbuffer := membershipstartDate.AddDate(0, 4, 7)

			if now.Before(onemonthplusbuffer) {

				refundamount, err := calculaterefundamount(ctx, membershipdetails.TokenDeposited, platinumlevel, "1")

				log.Printf("refund amount will be %v", refundamount)

				if err != nil {
					return "", fmt.Errorf("error: %v", err)
				}

				membershipdetails.RefundAmount = refundamount
				membershipdetails.IsCancelled = true
				membershipdetails.IsCompleted = true

				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				// transfer remaining tokens back to user
				err = h.Mint(ctx, refundamount)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				return "Successfully Cancel Membership", nil

			} else if now.Before(twomonthplusbuffer) {

				refundamount, err := calculaterefundamount(ctx, membershipdetails.TokenDeposited, platinumlevel, "2")
				if err != nil {
					return "", fmt.Errorf("error: %v", err)
				}
				log.Printf("refund amount will be %v", refundamount)
				membershipdetails.RefundAmount = refundamount
				membershipdetails.IsCancelled = true
				membershipdetails.IsCompleted = true

				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				// transfer remaining tokens back to user
				err = h.Mint(ctx, refundamount)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				return "Successfully Cancel Membership", nil

			} else if now.Before(threemonthplusbuffer) {

				refundamount, err := calculaterefundamount(ctx, membershipdetails.TokenDeposited, platinumlevel, "3")
				if err != nil {
					return "", fmt.Errorf("error: %v", err)
				}
				log.Printf("refund amount will be %v", refundamount)
				membershipdetails.RefundAmount = refundamount
				membershipdetails.IsCancelled = true
				membershipdetails.IsCompleted = true

				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				// transfer remaining tokens back to user
				err = h.Mint(ctx, refundamount)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				return "Successfully Cancel Membership", nil

			} else if now.Before(fourmonthplusbuffer) {

				refundamount, err := calculaterefundamount(ctx, membershipdetails.TokenDeposited, platinumlevel, "4")
				if err != nil {
					return "", fmt.Errorf("error: %v", err)
				}
				log.Printf("refund amount will be %v", refundamount)
				membershipdetails.RefundAmount = refundamount
				membershipdetails.IsCancelled = true
				membershipdetails.IsCompleted = true

				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				// transfer remaining tokens back to user
				err = h.Mint(ctx, refundamount)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				return "Successfully Cancel Membership", nil

			} else {
				return "", fmt.Errorf("cannot cancel Platinum memberhsip after 4 months")
			}

		} else {
			membershipstartDate, _ := time.Parse("01-02-2006", membershipdetails.StartDate)
			now := time.Now()
			// for testing purpose
			now = now.AddDate(0, 3, 0)

			onemonthplusbuffer := membershipstartDate.AddDate(0, 1, 7)
			twomonthplusbuffer := membershipstartDate.AddDate(0, 2, 7)
			threemonthplusbuffer := membershipstartDate.AddDate(0, 3, 7)
			fourmonthplusbuffer := membershipstartDate.AddDate(0, 4, 7)
			fivemonthplusbuffer := membershipstartDate.AddDate(0, 5, 7)
			sixmonthplusbuffer := membershipstartDate.AddDate(0, 6, 7)
			sevenmonthplusbuffer := membershipstartDate.AddDate(0, 7, 7)

			if now.Before(onemonthplusbuffer) {

				refundamount, err := calculaterefundamount(ctx, membershipdetails.TokenDeposited, diamondlevel, "1")
				if err != nil {
					return "", fmt.Errorf("error: %v", err)
				}
				log.Printf("refund amount will be %v", refundamount)
				membershipdetails.RefundAmount = refundamount
				membershipdetails.IsCancelled = true
				membershipdetails.IsCompleted = true

				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				// transfer remaining tokens back to user
				err = h.Mint(ctx, refundamount)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				return "Successfully Cancel Membership", nil

			} else if now.Before(twomonthplusbuffer) {

				refundamount, err := calculaterefundamount(ctx, membershipdetails.TokenDeposited, diamondlevel, "2")
				if err != nil {
					return "", fmt.Errorf("error: %v", err)
				}
				log.Printf("refund amount will be %v", refundamount)
				membershipdetails.RefundAmount = refundamount
				membershipdetails.IsCancelled = true
				membershipdetails.IsCompleted = true

				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				// transfer remaining tokens back to user
				err = h.Mint(ctx, refundamount)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				return "Successfully Cancel Membership", nil

			} else if now.Before(threemonthplusbuffer) {

				refundamount, err := calculaterefundamount(ctx, membershipdetails.TokenDeposited, diamondlevel, "3")
				if err != nil {
					return "", fmt.Errorf("error: %v", err)
				}
				log.Printf("refund amount will be %v", refundamount)
				membershipdetails.RefundAmount = refundamount
				membershipdetails.IsCancelled = true
				membershipdetails.IsCompleted = true

				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				// transfer remaining tokens back to user
				err = h.Mint(ctx, refundamount)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				return "Successfully Cancel Membership", nil

			} else if now.Before(fourmonthplusbuffer) {

				refundamount, err := calculaterefundamount(ctx, membershipdetails.TokenDeposited, diamondlevel, "4")
				if err != nil {
					return "", fmt.Errorf("error: %v", err)
				}
				log.Printf("refund amount will be %v", refundamount)
				membershipdetails.RefundAmount = refundamount
				membershipdetails.IsCancelled = true
				membershipdetails.IsCompleted = true

				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				// transfer remaining tokens back to user
				err = h.Mint(ctx, refundamount)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				return "Successfully Cancel Membership", nil

			} else if now.Before(fivemonthplusbuffer) {

				refundamount, err := calculaterefundamount(ctx, membershipdetails.TokenDeposited, diamondlevel, "5")
				if err != nil {
					return "", fmt.Errorf("error: %v", err)
				}
				log.Printf("refund amount will be %v", refundamount)

				membershipdetails.RefundAmount = refundamount
				membershipdetails.IsCancelled = true
				membershipdetails.IsCompleted = true

				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				// transfer remaining tokens back to user
				err = h.Mint(ctx, refundamount)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				return "Successfully Cancel Membership", nil

			} else if now.Before(sixmonthplusbuffer) {

				refundamount, err := calculaterefundamount(ctx, membershipdetails.TokenDeposited, diamondlevel, "6")
				if err != nil {
					return "", fmt.Errorf("error: %v", err)
				}
				log.Printf("refund amount will be %v", refundamount)

				membershipdetails.RefundAmount = refundamount
				membershipdetails.IsCancelled = true
				membershipdetails.IsCompleted = true

				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				// transfer remaining tokens back to user
				err = h.Mint(ctx, refundamount)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				return "Successfully Cancel Membership", nil

			} else if now.Before(sevenmonthplusbuffer) {

				refundamount, err := calculaterefundamount(ctx, membershipdetails.TokenDeposited, diamondlevel, "7")
				if err != nil {
					return "", fmt.Errorf("error: %v", err)
				}
				log.Printf("refund amount will be %v", refundamount)

				membershipdetails.RefundAmount = refundamount
				membershipdetails.IsCancelled = true
				membershipdetails.IsCompleted = true

				updatedmembership, _ := json.Marshal(membershipdetails)
				err = ctx.GetStub().PutState(currentmembershipId, updatedmembership)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				// transfer remaining tokens back to user
				err = h.Mint(ctx, refundamount)
				if err != nil {
					return "", fmt.Errorf("error:%v", err)
				}

				return "Successfully Cancel Membership", nil

			} else {
				return "", fmt.Errorf("cannot cancel Diamond memberhsip after 7 months")
			}
		}
	}
}

func calculaterefundamount(amount int, level string, month string) (int, error) {

	if level == platinumlevel {

		switch month {
		case "1":
			refundAmount := amount - ((amount * 20) / 100)
			return refundAmount, nil
		case "2":
			refundAmount := amount - ((amount * 40) / 100)
			return refundAmount, nil
		case "3":
			refundAmount := amount - ((amount * 60) / 100)
			return refundAmount, nil
		case "4":
			refundAmount := amount - ((amount * 80) / 100)
			return refundAmount, nil
		default:
			return 0, fmt.Errorf("??")
		}

	} else {

		switch month {
		case "1":
			refundAmount := amount - ((amount * 125) / 1000)
			return refundAmount, nil
		case "2":
			refundAmount := amount - ((amount * 25) / 100)
			return refundAmount, nil
		case "3":
			refundAmount := amount - ((amount * 375) / 1000)
			return refundAmount, nil
		case "4":
			refundAmount := amount - ((amount * 50) / 100)
			return refundAmount, nil
		case "5":
			refundAmount := amount - ((amount * 625) / 1000)
			return refundAmount, nil
		case "6":
			refundAmount := amount - ((amount * 75) / 100)
			return refundAmount, nil
		case "7":
			refundAmount := amount - ((amount * 875) / 1000)
			return refundAmount, nil
		default:
			return 0, fmt.Errorf("??")
		}
	}
}

func (h *HealthClub) GetAllMembershipsofUser(ctx contractapi.TransactionContextInterface) ([]string, error) {

	userid, err := ctx.GetClientIdentity().GetID()

	if err != nil {
		return nil, fmt.Errorf("failed to get userID: %v", err)
	}

	userId := userPrefix + userid

	user, err := ctx.GetStub().GetState(userId)
	if err != nil {
		return nil, fmt.Errorf("error:%v", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	userptr := new(User)
	_ = json.Unmarshal(user, &userptr)

	log.Printf("user details: %v", (userptr))

	if len(userptr.Memberships) == 0 {
		return nil, nil
	} else {
		return userptr.Memberships, nil
	}
}

func (h *HealthClub) GetMembershipDetails(ctx contractapi.TransactionContextInterface, membershipId string) (*Membership, error) {

	memberhsipbytes, err := ctx.GetStub().GetState(membershipId)
	if err != nil {
		return nil, fmt.Errorf("error:%v", err)
	}

	if memberhsipbytes == nil {
		return nil, fmt.Errorf("membership not found")
	}

	memberhsipdetails := new(Membership)
	_ = json.Unmarshal(memberhsipbytes, &memberhsipdetails)

	return memberhsipdetails, nil
}

func (h *HealthClub) GetUserDetails(ctx contractapi.TransactionContextInterface, userId string) (*User, error) {

	userbytes, err := ctx.GetStub().GetState(userId)
	if err != nil {
		return nil, fmt.Errorf("error:%v", err)
	}

	if userbytes == nil {
		return nil, fmt.Errorf("user not found")
	}

	userdetails := new(User)
	_ = json.Unmarshal(userbytes, &userdetails)

	return userdetails, nil
}

func (h *HealthClub) GetLevelDetails(ctx contractapi.TransactionContextInterface, level string) (*Level, error) {

	levelbytes, err := ctx.GetStub().GetState(level)
	if err != nil {
		return nil, fmt.Errorf("error:%v", err)
	}

	if levelbytes == nil {
		return nil, fmt.Errorf("level not found")
	}

	leveldetails := new(Level)
	_ = json.Unmarshal(levelbytes, &leveldetails)

	return leveldetails, nil
}

func (h *HealthClub) UpgradeMembership(ctx contractapi.TransactionContextInterface, level string) (string, error) {

	// get unique user id
	userId, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}

	if userId == "" {
		return "", fmt.Errorf("userID not exist")
	}

	userDetails := new(User)
	resInBytes1, err := ctx.GetStub().GetState(userPrefix + userId)
	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}

	if resInBytes1 == nil {
		return "", fmt.Errorf("empty userDetails")
	}
	_ = json.Unmarshal(resInBytes1, &userDetails)

	level_ := new(Level)
	resInBytes2, err := ctx.GetStub().GetState(level)
	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}
	if resInBytes1 == nil {
		return "", fmt.Errorf("empty level")
	}
	_ = json.Unmarshal(resInBytes2, &level_)

	membership := new(Membership)

	a := len(userDetails.Memberships)

	currentMembershipID := userDetails.Memberships[a-1]

	resInBytes3, err := ctx.GetStub().GetState(currentMembershipID)

	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}
	_ = json.Unmarshal(resInBytes3, &membership)

	if err != nil {
		return "", fmt.Errorf("error:::%v", err.Error())
	}

	oldLevelcopy := membership.Level

	currentTime := time.Now()

	endTime, _ := time.Parse("01-02-2006", membership.EndDate)

	checkExpire := (endTime.Sub(currentTime)).Hours()

	if level != goldlevel && level != diamondlevel && level != platinumlevel {
		return "", fmt.Errorf("level not matched,expected Gold,Platinum,Diamond but given: %v", level)
	}

	if int(checkExpire) <= 0 {
		return "", fmt.Errorf("un-expected time difference: %v", checkExpire)
	}

	if checkExpire <= 24 {
		return "", fmt.Errorf("membership expired at %v cannot update", checkExpire)
	}

	months := (checkExpire / 730)

	if int(months) >= level_.Months {
		return "", fmt.Errorf("cannot de-grade membership from %v to %v", membership.Level, level)
	}

	tokens := level_.EntryPrizeTokens - membership.TokenDeposited

	if tokens <= 0 {
		return "", fmt.Errorf("already on %v level", level)
	}

	newMonths := int(months) + (level_.Months - 1)
	membership.EndDate = currentTime.AddDate(0, int(newMonths), 0).Format("01-02-2006")
	membership.TokenDeposited = tokens + membership.TokenDeposited
	membership.IsCompleted = false
	membership.IsUpdated = true
	membership.Level = level

	resInBytes, _ := json.Marshal(membership)

	if err != nil {
		return "", fmt.Errorf("err: %v", err)
	}

	err = ctx.GetStub().PutState(currentMembershipID, resInBytes)
	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}

	userdetailsbytes, _ := json.Marshal(userDetails)
	err = ctx.GetStub().PutState(userPrefix+userId, userdetailsbytes)
	if err != nil {
		return "", fmt.Errorf("error:%v", err)
	}

	checkForCompositeKey, err := h.GetParticularMemberByLevel(ctx, level, userId)

	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}
	var index string = "level~UserID"
	if !checkForCompositeKey {

		userLevelIndexKey, err := ctx.GetStub().CreateCompositeKey(index, []string{level, userId})
		if err != nil {
			return "", fmt.Errorf("failed to create the composite key for prefix %s: %v", index, err)
		}

		err = ctx.GetStub().PutState(userLevelIndexKey, userdetailsbytes)
		if err != nil {
			return "", fmt.Errorf("error %v", err)
		}
	}

	checkForCompositeKey, err = h.GetParticularMemberByLevel(ctx, oldLevelcopy, userId)

	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}

	if checkForCompositeKey {

		userLevelIndexKey, err := ctx.GetStub().CreateCompositeKey(index, []string{oldLevelcopy, userPrefix + userId})
		if err != nil {
			return "", fmt.Errorf("failed to create the composite key for prefix %s: %v", index, err)
		}
		err = ctx.GetStub().DelState(userLevelIndexKey)
		if err != nil {
			return "", fmt.Errorf("error in del state for %v level composite key", oldLevelcopy)
		}
	}

	log.Printf("membership updated from %v to %v at %v tokens", membership.StartDate, membership.EndDate, membership.TokenDeposited)

	adminidbytes, err := ctx.GetStub().GetState("owner")

	if err != nil {
		return "", fmt.Errorf("err: %v", err)
	}

	if adminidbytes == nil {
		return "", fmt.Errorf("AdminID not set")
	}

	adminID := string(adminidbytes)

	err = h.Transfer(ctx, adminID, tokens)

	if err != nil {
		return "", fmt.Errorf("err: %v", err)
	}

	return "Membership Updated", nil

}

func (h *HealthClub) GetParticularMemberByLevel(ctx contractapi.TransactionContextInterface, level string, userId string) (bool, error) {
	var index string = "level~UserID"
	var resultBool bool
	//return bytes memory address of first state
	levelIterator, err := ctx.GetStub().GetStateByPartialCompositeKey(index, []string{level})
	if err != nil {
		return false, fmt.Errorf("error:%v", err.Error())
	}

	defer levelIterator.Close()

	// Iterate through result set and for each level found
	var i int
	for i = 0; levelIterator.HasNext(); i++ {
		responseRange, err := levelIterator.Next()
		if err != nil {
			return false, fmt.Errorf("error:%v", err.Error())
		}
		// get the level and userID from defined index composite key
		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(responseRange.Key)
		if err != nil {
			return false, fmt.Errorf("error:%v", err.Error())
		}
		returnedLevelUserID := compositeKeyParts[1]
		if returnedLevelUserID == userPrefix+userId {
			resultBool = true
		} else {
			resultBool = false
		}
	}
	return resultBool, nil
}

func (h *HealthClub) GetAllMembershipsOfUsers(ctx contractapi.TransactionContextInterface) ([]Membership, error) {

	membership := new(Membership)

	arr := []Membership{}

	resInBytes, err := ctx.GetStub().GetState("TotalMemberships")
	breakingPoint, _ := strconv.Atoi(string(resInBytes))

	if err != nil {
		return nil, fmt.Errorf("error1:%v", err.Error())
	}

	for i := 0; i <= breakingPoint; i++ {

		key := membershipPrefix + strconv.Itoa(i)
		resInBytes, err = ctx.GetStub().GetState(key)

		if err != nil {
			return nil, fmt.Errorf("error:%v", err.Error())
		}

		if resInBytes == nil {

		}

		_ = json.Unmarshal(resInBytes, &membership)

		arr = append(arr, *membership)

	}
	return arr[1:], nil
}

func (h *HealthClub) GetAllUsers(ctx contractapi.TransactionContextInterface) ([]User, error) {
	startKey := ""
	endKey := ""

	resultsIterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	log.Print("resultsIterator", resultsIterator)

	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	results := []User{}

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		checkPrefix := strings.HasPrefix(queryResponse.Key, userPrefix)
		if checkPrefix == true {
			club := new(User)
			_ = json.Unmarshal(queryResponse.Value, club)
			results = append(results, *club)

		}

	}

	return results, nil
}

func (h *HealthClub) UpdateMembershipLevelToken(ctx contractapi.TransactionContextInterface, level string, entryPrizeTokens int) (string, error) {

	level_ := new(Level)

	userId, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}

	adminidbytes, err := ctx.GetStub().GetState("owner")

	if err != nil {
		return "", fmt.Errorf("err: %v", err)
	}

	if adminidbytes == nil {
		return "", fmt.Errorf("adminID not set")
	}

	adminID := string(adminidbytes)

	if adminID != userId {
		return "", fmt.Errorf("only owner can update level tokens")
	}

	resInBytes, err := ctx.GetStub().GetState(level)

	if err != nil {
		return "", fmt.Errorf("err: %v", err)
	}

	if resInBytes == nil {
		return "", fmt.Errorf("levl is emplty")
	}

	_ = json.Unmarshal(resInBytes, level_)

	if err != nil {
		return "", fmt.Errorf("err: %v", err)
	}

	if level_.EntryPrizeTokens == entryPrizeTokens {
		return "", fmt.Errorf("no unique value given")
	}

	if level == goldlevel || level == diamondlevel || level == platinumlevel {

		level_.EntryPrizeTokens = entryPrizeTokens

		resInBytes, err := json.Marshal(level_)
		if err != nil {
			return "", fmt.Errorf("error:%v", err)
		}

		err = ctx.GetStub().PutState(level, resInBytes)
		if err != nil {
			return "", fmt.Errorf("error:%v", err)
		}

		log.Printf("The %v level is updated at %v tokens ", level, entryPrizeTokens)

	} else {
		return "", fmt.Errorf("only Gold, Diamond, and Platinum levels are acceptable")
	}

	return "level is updated", nil
}
