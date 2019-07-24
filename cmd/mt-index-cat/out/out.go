package out

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/davecgh/go-spew/spew"
	"github.com/grafana/metrictank/idx"
)

var QueryTime int64

func Dump(d idx.MetricDefinitionInterned) {
	spew.Dump(d.ConvertToSchemaMd())
}

func List(d idx.MetricDefinitionInterned) {
	fmt.Println(d.OrgId, d.NameWithTags())
}

func GetVegetaRender(addr, from string) func(d idx.MetricDefinitionInterned) {
	return func(d idx.MetricDefinitionInterned) {
		fmt.Printf("GET %s/render?target=%s&from=-%s\nX-Org-Id: %d\n\n", addr, d.Name.String(), from, d.OrgId)
	}
}

func GetVegetaRenderPattern(addr, from string) func(d idx.MetricDefinitionInterned) {
	return func(d idx.MetricDefinitionInterned) {
		fmt.Printf("GET %s/render?target=%s&from=-%s\nX-Org-Id: %d\n\n", addr, pattern(d.Name.String()), from, d.OrgId)
	}
}

func Template(format string) func(d idx.MetricDefinitionInterned) {
	funcs := make(map[string]interface{})
	funcs["pattern"] = pattern
	funcs["patternCustom"] = patternCustom
	funcs["age"] = age
	funcs["roundDuration"] = roundDuration

	// replace '\n' in the format string with actual newlines.
	format = strings.Replace(format, "\\n", "\n", -1)

	tpl := template.Must(template.New("format").Funcs(funcs).Parse(format))

	return func(d idx.MetricDefinitionInterned) {
		md := d.ConvertToSchemaMd()
		err := tpl.Execute(os.Stdout, &md)
		if err != nil {
			panic(err)
		}
	}
}
