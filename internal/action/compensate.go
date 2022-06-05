package action

type Compensate func() error

func (c Compensate) Run() error {
	if c == nil {
		return nil
	}
	return c()
}

func (c Compensate) With(other Compensate) Compensate {
	if other == nil {
		return c
	}
	if c == nil {
		return other
	}
	return func() error {
		if err := other(); err != nil {
			return err
		}
		return c()
	}
}

// func RunCompensate(originalErr error, c Compensate) error {
// 	if c == nil {
// 		return originalErr
// 	}
// 	compensateErr := c()
// 	if compensateErr != nil {
// 		compensateErr = fmt.Errorf("undoing actions: %w", compensateErr)
// 	}
// 	return JoinErrors(originalErr, compensateErr)
// }

// func JoinErrors(errs ...error) error {
// 	me := multiError{errs: make([]error, 0, len(errs))}
// 	for _, err := range errs {
// 		if err == nil {
// 			continue
// 		} else if m, ok := err.(multiError); ok {
// 			me.errs = append(me.errs, m.errs...)
// 		} else {
// 			me.errs = append(me.errs, err)
// 		}
// 	}
// 	if len(me.errs) == 0 {
// 		return nil
// 	}
// 	return me
// }

// type multiError struct {
// 	errs []error
// }

// func (me multiError) Unwrap() error {
// 	if len(me.errs) != 1 {
// 		return nil
// 	}
// 	return me.errs[0]
// }

// func (me multiError) Error() string {
// 	if len(me.errs) == 0 {
// 		return "unknown error"
// 	}
// 	if len(me.errs) == 1 {
// 		return me.errs[0].Error()
// 	}

// 	b := strings.Builder{}
// 	b.WriteString("following failures occurred:")
// 	for _, e := range me.errs {
// 		if e == nil {
// 			continue
// 		}
// 		b.WriteString("\n  ")
// 		b.WriteString(e.Error())
// 	}
// 	return b.String()
// }
