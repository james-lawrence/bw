package httputilx

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/gorilla/mux"
)

type routeInfo struct {
	file, name, route string
}

type routes []routeInfo

func (t routes) Len() int      { return len(t) }
func (t routes) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

type byRoute struct{ routes }

func (t byRoute) Less(i, j int) bool { return t.routes[i].route < t.routes[j].route }

// Printer prints all the routes on the router.
func Printer(router *mux.Router) {
	maxFileLength := 25
	maxNameLength := 25
	maxRouteLength := 25

	routesInfo := make([]routeInfo, 0, 10)
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		max := func(a, b int) int {
			if a > b {
				return a
			}
			return b
		}

		routeName := func(s string) string {
			if strings.TrimSpace(s) != "" {
				return s
			}
			return "unnamed route"
		}

		routeURI := func(s string, err error) string {
			if err == nil {
				return s
			}

			return fmt.Sprintf("unknown path (%s)", err)
		}

		handlerLocation := func(h http.Handler) string {
			if h != nil {
				f := runtime.FuncForPC(reflect.ValueOf(h).Pointer())
				file, line := f.FileLine(f.Entry())
				return fmt.Sprintf("%s:%d", filepath.Base(file), line)
			}

			return "no associated handler"
		}

		if route.GetHandler() == nil {
			return nil
		}

		i := routeInfo{
			file:  handlerLocation(route.GetHandler()),
			name:  routeName(route.GetName()),
			route: routeURI(route.GetPathTemplate()),
		}
		maxFileLength = max(maxFileLength, len(i.file))
		maxNameLength = max(maxNameLength, len(i.name))
		maxRouteLength = max(maxRouteLength, len(i.route))
		routesInfo = append(routesInfo, i)
		return nil
	})

	pattern := fmt.Sprintf("%%-%d.%ds|%%-%d.%ds|%%-%d.%ds\n", maxFileLength, maxFileLength, maxNameLength, maxNameLength, maxRouteLength, maxRouteLength)
	log.Printf(pattern, "file", "name", "route")

	sort.Sort(byRoute{routesInfo})

	for _, info := range routesInfo {
		log.Printf(pattern, info.file, info.name, info.route)
	}
}
