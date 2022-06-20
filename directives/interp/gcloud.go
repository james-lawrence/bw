package interp

import (
	"context"
	"reflect"

	"github.com/james-lawrence/bw/directives/interp/gcloud"
	"github.com/james-lawrence/bw/internal/x/errorsx"
)

func gcloudtargetpool() (exported map[string]reflect.Value) {
	restart := func(ctx context.Context, do func(context.Context) error) (err error) {
		return errorsx.Compact(
			gcloud.TargetPoolDetach(ctx),
			do(ctx),
			gcloud.TargetPoolAttach(ctx),
		)
	}

	exported = map[string]reflect.Value{
		"Restart": reflect.ValueOf(restart),
	}

	return exported
}
