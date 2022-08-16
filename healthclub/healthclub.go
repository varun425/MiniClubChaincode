func (h *HealthClub) update(ctx contractapi.TransactionContextInterface, level string, tokens int) (string, error) {

	// get unique user id
	userId, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}

	userDetails := new(User)
	resInBytes, err := ctx.GetStub().GetState("User-" + userId)
	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}
	json.Unmarshal(resInBytes, userDetails)

	level_ := new(Level)
	resInBytes, err = ctx.GetStub().GetState(level)
	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}
	json.Unmarshal(resInBytes, level_)

	//nMT - oMT = n token
	membership := new(Membership)

	a := len(userDetails.Memberships)
	resInBytes, err = ctx.GetStub().GetState(membershipPrefix + strconv.Itoa(a-1))

	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}
	json.Unmarshal(resInBytes, membership)
	currentTime := time.Now()
	endTime, _ := time.Parse("01-02-2006", membership.EndDate)
	checkExpire := (endTime.Sub(currentTime).Hours())

	if int(checkExpire) <= 0 {
		return "", fmt.Errorf("un-expected time difference: %v", checkExpire)
	}

	if checkExpire <= 24 {
		return "", fmt.Errorf("membership expired at %v cannot update", checkExpire)
	}

	months := (checkExpire / 730)
	//6 > 5
	if int(months) >= level_.Months {
		return "", fmt.Errorf("cannot de-grade membership from %v to %v", level_.Months, int(months))
	}

	n := level_.EntryPrizeTokens - membership.TokenDeposited
	if n != tokens {
		return "", fmt.Errorf("required token = %v but given = %v", n, tokens)
	}

	newMonths := int(months) + (level_.Months - 1)
	membership.StartDate = currentTime.Format("01-02-2006")
	membership.EndDate = currentTime.AddDate(0, int(newMonths), 0).Format("01-02-2006")
	membership.TokenDeposited = tokens + membership.TokenDeposited

	resInBytes, _ = json.Marshal(membership)
	//membershipID := membershipPrefix + strconv.Itoa(TotalMemberships+1)
	err = ctx.GetStub().PutState(membershipPrefix+strconv.Itoa(a-1), resInBytes)
	if err != nil {
		return "", fmt.Errorf("error:%v", err.Error())
	}

	log.Printf("membership updated from %v to %v at %v tokens", membership.StartDate, membership.EndDate, membership.TokenDeposited)

	return "", nil

}
