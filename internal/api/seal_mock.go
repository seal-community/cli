//go:build mockserver
// +build mockserver

package api

// needs to be var to inject port string during build
var serverPort string = ""

var BaseURL = "http://127.0.0.1:" + serverPort + "/cli.sealsecurity.io"

var AuthURL = "http://127.0.0.1:" + serverPort + "/authorization.sealsecurity.io"
var PypiServer = "http://127.0.0.1:" + serverPort + "/pypi.sealsecurity.io"
var NpmServer = "http://127.0.0.1:" + serverPort + "/npm.sealsecurity.io"
var NugetServer = "http://127.0.0.1:" + serverPort + "/nuget.sealsecurity.io"
var MavenServer = "http://127.0.0.1:" + serverPort + "/maven.sealsecurity.io"
var GolangServer = "http://127.0.0.1:" + serverPort + "/go.sealsecurity.io"
var PackagistServer = "http://127.0.0.1:" + serverPort + "/packagist.sealsecurity.io"
var RpmServer = "http://127.0.0.1:" + serverPort + "/rpm.sealsecurity.io"
