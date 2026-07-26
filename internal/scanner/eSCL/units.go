package escl

// escl units represent 1/300 of an inch
// using micrometres internally instead of millimetres to avoid precision issues

func ToEsclUnits(micrometres int) int {
	return (micrometres*3 + 127) / 254
}

func ToMicrometres(esclUnits int) int {
	return (esclUnits*254 + 127) / 3
}
