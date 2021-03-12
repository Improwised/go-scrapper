package spiders

type RootSpider interface {
	Run()
	SetAdditionalArgs(args string)
}

