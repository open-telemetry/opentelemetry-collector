package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/jzelinskie/must"
)

func findAllBenchmarks(b []byte) []string {
	arr := bytes.SplitN(b, []byte("profiles = map[string]*pprof.Profile{"), 2)
	arr2 := bytes.SplitN(arr[1], []byte("}"), 2)

	r := regexp.MustCompile("\"(.+?)\":")
	all := r.FindAllStringSubmatch(string(arr2[0]), -1)
	res := make([]string, len(all))
	for i, v := range all {
		res[i] = v[1]
	}
	// log.Println(all[1])
	return res
}

var tmpltString = `
func Benchmark{{.CapitalizedName}}Pprof(b *testing.B) {
	benchmark(b, "{{.Name}}", "pprof")
}
func Benchmark{{.CapitalizedName}}Denormalized(b *testing.B) {
	benchmark(b, "{{.Name}}", "denormalized")
}
func Benchmark{{.CapitalizedName}}Normalized(b *testing.B) {
	benchmark(b, "{{.Name}}", "normalized")
}
func Benchmark{{.CapitalizedName}}Arrays(b *testing.B) {
	benchmark(b, "{{.Name}}", "arrays")
}
func Benchmark{{.CapitalizedName}}PprofExtended(b *testing.B) {
	benchmark(b, "{{.Name}}", "pprofextended")
}
`

func main() {
	tmpl := must.NotError(template.New("test").Parse(tmpltString))

	f := must.NotError(os.ReadFile("benchmark_test.go"))
	arr := bytes.SplitN(f, []byte("// benchmarks"), 2)
	res := string(arr[0])
	res += "// benchmarks\n"
	res += "//\n"

	benchmarks := findAllBenchmarks(f)
	log.Println(benchmarks)
	buf := &bytes.Buffer{}

	for _, b := range benchmarks {
		name := string(b)
		name = strings.ToUpper(string(name[0])) + name[1:]
		tmpl.Execute(buf, struct {
			CapitalizedName string
			Name            string
		}{CapitalizedName: name, Name: string(b)})
	}

	res += buf.String()

	fmt.Println(res)

	os.WriteFile("benchmark_test.go", []byte(res), 0644)

	// res := string(arr[0])

}
