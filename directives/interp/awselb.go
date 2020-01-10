package interp

import (
	"context"
	"reflect"

	"github.com/james-lawrence/bw/directives/awselb2"
)

func elb() (exported map[string]reflect.Value) {
	restart := func(ctx context.Context, do func(context.Context) error) (err error) {
		if err = awselb2.LoadbalancersDetach(ctx); err != nil {
			return err
		}

		if err = do(ctx); err != nil {
			return err
		}

		return awselb2.LoadbalancersAttach(ctx)
	}

	exported = map[string]reflect.Value{
		"Restart": reflect.ValueOf(restart),
	}

	return exported
}
