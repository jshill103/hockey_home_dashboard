package services

// normalizeWinningScore guarantees the displayed final score matches the chosen winner.
func normalizeWinningScore(homeGoals, awayGoals int, winner, homeTeam, awayTeam string) (int, int) {
	if homeGoals < 0 {
		homeGoals = 0
	}
	if awayGoals < 0 {
		awayGoals = 0
	}

	switch winner {
	case homeTeam:
		if homeGoals <= awayGoals {
			homeGoals = awayGoals + 1
		}
	case awayTeam:
		if awayGoals <= homeGoals {
			awayGoals = homeGoals + 1
		}
	default:
		if homeGoals == awayGoals {
			homeGoals++
		}
	}

	return homeGoals, awayGoals
}
