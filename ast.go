package main

type Operator int

const (
	Equal Operator = iota + 1
	Replace
	Merge
)

type Step struct {
	Index      *int    `json:"idx,omitempty"`
	Identifier *string `json:"ident,omitempty"`
}

type Argument struct {
	Number *int    `json:"num,omitempty"`
	String *string `json:"string,omitempty"`
	Path   []Step  `json:"path,omitempty"`
}

type Expression struct {
	Operator  Operator   `json:"op,omitempty"`
	Arguments []Argument `json:"args"`
}

type Node struct {
	Expression Expression `json:"expression"`
	Pipe       *Node      `json:"pipe,omitempty"`
}

func (a *Argument) AppendStepToPath(step Step) {
	if a.Path == nil {
		a.Path = []Step{}
	}
	a.Path = append(a.Path, step)
}
