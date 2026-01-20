package audit

type Observer interface {
	Notify(event Event) error
}
