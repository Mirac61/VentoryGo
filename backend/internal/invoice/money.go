package invoice

type Money int64

func RoundedVAT(net Money, rateBasisPoints int64) Money {
	return Money((int64(net)*rateBasisPoints + 5000) / 10000)
}
