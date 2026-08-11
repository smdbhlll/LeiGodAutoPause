package processes

type Process struct {
	PID  uint32 `json:"pid"`
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

type Scanner interface {
	List() ([]Process, error)
}

func NewScanner() Scanner { return nativeScanner{} }
