package temperature

type (
	// Celsius is a temperature in C
	Celsius float64

	// Kelvin is a temperature in K
	Kelvin float64

	// Fahrenheit is a temperature in deg F
	Fahrenheit float64
)

// C2F converts a temp in Celsius to Fahrenheit
func C2F(c Celsius) Fahrenheit {
	return Fahrenheit(c*9/5 + 32)
}

// C2K converts a temp in Celsius to Kelvin
func C2K(c Celsius) Kelvin {
	return Kelvin(c + 273.15)
}

// K2C converts a temp in Kelvin to Celsius
func K2C(k Kelvin) Celsius {
	return Celsius(k - 273.15)
}

// F2C converts a temp in Fahrenheit to Celcius
func F2C(f Fahrenheit) Celsius {
	return Celsius((f - 32) * 5 / 9)
}

// F2K converts a temp in Fahrenheit to Kelvin
func F2K(f Fahrenheit) Kelvin {
	c := F2C(f)
	return C2K(c)
}
