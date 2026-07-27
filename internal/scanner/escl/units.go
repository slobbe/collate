package escl

// escl units represent 1/300 of an inch
// using micrometres internally instead of millimetres to avoid precision issues

func toESCLUnits(micrometres int) int {
	return (micrometres*3 + 127) / 254
}

func toMicrometres(esclUnits int) int {
	return (esclUnits*254 + 1) / 3
}
