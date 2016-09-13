package main

import "fmt"
import "github.com/balzaczyy/golucene/core/util"
import "os"
import "github.com/balzaczyy/golucene/core/index"
import "github.com/balzaczyy/golucene/core/search"
import "github.com/balzaczyy/golucene/core/store"
import std "github.com/balzaczyy/golucene/analysis/standard"

func main() {
	fmt.Println("Wefwef")
	util.SetDefaultInfoStream(util.NewPrintStreamInfoStream(os.Stdout))
	index.DefaultSimilarity = func() index.Similarity {
		return search.NewDefaultSimilarity()
	}

	directory, _ := store.OpenFSDirectory("test_index")
	analyzer := std.NewStandardAnalyzer()
	conf := index.NewIndexWriterConfig(util.VERSION_LATEST, analyzer)
	writer, _ := index.NewIndexWriter(directory, conf)
}
