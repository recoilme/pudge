package pudge

func ExampleSet() {
	Set("test/test", "Hello", "World")
	defer CloseAll()
}
