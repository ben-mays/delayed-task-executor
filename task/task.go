package task

type Task interface {
	Run() (int, error)
}
