# SPA Serve

[![Go Reference](https://pkg.go.dev/badge/github.com/thediveo/spaserve.svg)](https://pkg.go.dev/github.com/thediveo/spaserve)
![GitHub](https://img.shields.io/github/license/thediveo/spaserve)
![build and test](https://github.com/TheDiveO/spaserve/workflows/build%20and%20test/badge.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thediveo/spaserve)](https://goreportcard.com/report/github.com/thediveo/spaserve)
![Coverage](https://img.shields.io/badge/Coverage-95.5%25-brightgreen)

`spaserve` serves "Single Page Applications" (SPAs) from Go that are using...

- ...client-side DOM routing,
- ...varying base paths in different deployments or even within the same
  deployment because of multiple access paths.

And all this **without the need to rebuild your SPA production code** just
because the (HTML) "[base
URL](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/base)" changes.

## Usage

1. prepare your SPA `index.html` template (such as `public/index.html`) by
   adding a `base` element, if not done already; set its `href` attribute to
   `./`:

   ```html
   <!doctype html>
   <html lang="en">
   <head>
       <base href="./" />
       <!-- ... -->
    </head>
    <body>
        <!-- ... -->
    </body>
    </html>
   ```

2. In case of CRA (Create React App), set the `homepage` field in `package.json`
   to `"."` (**not** the root slash ~~`"/"`~~):

   ```json
   {
     "homepage": ".",
   }
   ```

3. add a basename helper to your SPA sources, such as a new file
   `src/util/basename.ts`:

   ```ts
   export const basename = new URL(
           ((document.querySelector('base') || {}).href || '/')
       ).pathname.replace(/\/$/, '')
   ```

4. in your `App.tsx` ensure that you tell your client-side DOM router to
   correctly pick up the basename; this makes reloading the SPA from any route
   and bookmarking routes possible:

   ```tsx
   import { basename } from 'utils/basename'

   <Router basename={basename}>
       <!-- your app components here -->
   </Router>
   ```

5. **Make sure that all links (and asset references) are relative**, such as
   `./view2`, et cetera.

6. in your service, create your HTTP route muxer and set up your API handlers as
   usual, then create a `SPAHandler` and register it as the route handler to be
   used when all other handlers don't match:

   ```go
   r := mux.NewRouter() // or whatever you prefer
   // (set up all your API routes)

   // finally create a suitable fs.FS to be used with the SPAHandler
   // and register it so that it serves on all routes not handled by
   // the more specific (API) handlers. Here, we assume the SPA assets
   // to be rooted in web/build.
   spa := spaserve.NewSPAHandler(os.DirFS("web/build"), "index.html")
   r.PathPrefix("/").Handler(spa)
   ```

## References

Useful background knowledge when dealing with serving HTTP resources,
base(names), et cetera...

- [An elegant solution of deploying React app into a
  subdirectory](https://skryvets.com/blog/2018/09/20/an-elegant-solution-of-deploying-react-app-into-a-subdirectory/)
  (_Sergey Kryvets_) – a rare competent analysis and introduction to the `base`
  HTML element. This post shows how to get SPAs working with `base` for a
  _fixed_, _hardcoded_ base path. In contrast, `spaserver` _dynamically_
  rewrites the `base` element when serving an SPA, as needed.

- [answer to "_Golang. What to use? http.ServeFile(..) or
  http.FileServer(..)?_"](https://stackoverflow.com/a/28798174/6632214)
  (stackoverflow) – and yes, `spaserve` uses `http.FileServer` which supports
  `fs.FS` via an `http.FS` adaptor.

## Go Version Support

`spaserve` supports versions of Go that are noted by the [Go release
policy](https://golang.org/doc/devel/release.html#policy), that is, major
versions _N_ and _N_-1 (where _N_ is the current major version).

## Copyright and License

`spaserve` is Copyright 2022-23 Harald Albrecht, and licensed under the Apache
License, Version 2.0.
