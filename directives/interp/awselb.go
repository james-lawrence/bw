package interp

import (
	"context"
	"log"
	"reflect"

	"github.com/james-lawrence/bw/directives/awselb"
)

func elb() (exported map[string]reflect.Value) {
	restart := func(ctx context.Context, do func(context.Context) error) (err error) {
		if err = awselb.LoadbalancersDetach(ctx); err != nil {
			log.Printf("deteach failed %T - %+v\n", err, err)
			return err
		}

		if err = do(ctx); err != nil {
			return err
		}

		if err = awselb.LoadbalancersAttach(ctx); err != nil {
			log.Printf("attach failed %T - %+v\n", err, err)
			return err
		}

		return nil
	}

	exported = map[string]reflect.Value{
		"Restart": reflect.ValueOf(restart),
	}

	return exported
}
