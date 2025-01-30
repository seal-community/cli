//go:build mockserver
// +build mockserver

package api

// using vars to inject string values during build
var serverPort string = ""
var serverHost string = ""
var serverScheme string = ""

var mockServer = serverScheme + "://" + serverHost + ":" + serverPort

var BaseURL = mockServer + "/cli.sealsecurity.io"
var AuthURL = mockServer + "/authorization.sealsecurity.io"
var PypiServer = mockServer + "/pypi.sealsecurity.io"
var NpmServer = mockServer + "/npm.sealsecurity.io"
var NugetServer = mockServer + "/nuget.sealsecurity.io"
var MavenServer = mockServer + "/maven.sealsecurity.io"
var GolangServer = mockServer + "/go.sealsecurity.io"
var PackagistServer = mockServer + "/packagist.sealsecurity.io"
var RpmServer = mockServer + "/rpm.sealsecurity.io"
var DebServer = mockServer + "/deb.sealsecurity.io"
var ApkServer = mockServer + "/apk.sealsecurity.io"
