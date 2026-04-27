package auction

func SelectWinner(bids []Bid) (Bid, bool) {
	var winner Bid
	found := false
	maxScore := 0.0

	for _, bid := range bids {
		if !bid.Accepted {
			continue
		}

		score := bid.Score()
		if !found || score > maxScore {
			winner = bid
			maxScore = score
			found = true
		}
	}

	return winner, found
}
