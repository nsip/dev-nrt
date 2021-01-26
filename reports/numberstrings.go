package reports

//
// converting string to float to zero is too risky
// so simply test the number against the known possible
// zero representations as a string
//
func nonzero(number string) bool {

	switch {
	case number == "0", number == "0.0", number == "0.00", number == "0.000", number == "00.00":
		return false
	default:
		return true
	}

}
