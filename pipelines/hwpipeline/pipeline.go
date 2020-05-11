package hwpipeline

// HelloWorld ...
type HelloWorld struct {
	Message string
}

func (h HelloWorld) clone() HelloWorld {
	return HelloWorld{Message: h.Message}
}
