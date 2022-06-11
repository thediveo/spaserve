/*

Package spaserve serves "Single Page Applications" (SPAs), supporting
client-side DOM routing and varying base paths. And all this without the need to
rebuild the SPA production code when the deployment changes.

The SPAHandler type implements http.Handler to serve the SPA and its static
resources. The SPAHandler fetches these resources from any resource provider
implementing the fs.FS interface. This design even allows to seamlessly embed an
SPA into a Go binary.

*/
package spaserve
