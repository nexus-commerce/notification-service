package notification

const redacted = "[redacted]"

type RevealedToken struct {
	value string
}

func (t RevealedToken) Value() string { return t.value }

func (RevealedToken) String() string { return redacted }

func (RevealedToken) GoString() string { return redacted }
