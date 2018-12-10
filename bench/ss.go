package ss

type SS struct {
	keys  [][][]byte
	index [][]byte
}

func New() *SS {
	ss := new(SS)
	ss.keys = make([][][]byte, 0)
	ss.index = make([][]byte, 0)
	return ss
}

func (ss *SS) Set(b []byte) {
	ss.index = append(ss.index, b)
	ss.index = append(ss.index, b)

}
