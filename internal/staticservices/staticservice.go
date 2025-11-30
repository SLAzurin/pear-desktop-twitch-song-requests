package staticservices

type staticService interface {
	TestConnection() error
}
