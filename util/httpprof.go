package util

import (
	"net/http/pprof"
	"github.com/julienschmidt/httprouter"
	"fmt"
	"net/http"
)


func StartPProf(router *httprouter.Router){
	router.GET("/debug/pprof/", IndexHandler)
	router.GET("/debug/pprof/heap", HeapHandler)
	router.GET("/debug/pprof/goroutine", GoroutineHandler)
	router.GET("/debug/pprof/block", BlockHandler)
	router.GET("/debug/pprof/threadcreate", ThreadCreateHandler)
	router.GET("/debug/pprof/cmdline", CmdlineHandler)
	router.GET("/debug/pprof/profile", ProfileHandler)
	router.GET("/debug/pprof/symbol", SymbolHandler)
	
	fmt.Println("pprof url:\n","/debug/pprof/\n","/debug/pprof/heap\n","/debug/pprof/goroutine\n","/debug/pprof/block\n",
	"/debug/pprof/threadcreate\n","/debug/pprof/cmdline\n","/debug/pprof/profile\n","/debug/pprof/symbol\n")
}


func IndexHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params){
	pprof.Index(response,request)
}

func HeapHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params){
	pprof.Handler("heap").ServeHTTP(response, request)
}

func GoroutineHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params){
	pprof.Handler("goroutine").ServeHTTP(response, request)
}

func BlockHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params){
	pprof.Handler("block").ServeHTTP(response, request)
}
func ThreadCreateHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params){
	pprof.Handler("threadcreate").ServeHTTP(response, request)
}
func CmdlineHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params){
	pprof.Cmdline(response, request)
}
func ProfileHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params){
	pprof.Profile(response, request)
}
func SymbolHandler(response http.ResponseWriter, request *http.Request, ps httprouter.Params){
	pprof.Symbol(response, request)
}