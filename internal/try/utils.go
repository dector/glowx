package try

func Try(err error) {
	if err != nil {
		panic(err)
	}
}
