# Flux v2 Documentation

The documentation for `flux2` has moved to this repository: <https://github.com/fluxcd/website>.

[The Website's README](https://github.com/fluxcd/website/#readme) has information on how to

- modify and extend documentation
- run the site <https://fluxcd.io> locally
- publish changes

and where all the individual pieces of content come from.

It will be easier for us to maintain a coherent web presences (and merge all of Flux documentation) in one central repository. This was partly discussed in <https://github.com/fluxcd/flux2/discussions/367>.

## toolkit.fluxcd.io

For historical reasons we are keeping a `_redirects` file in this directory. It defines how redirects from the old site `toolkit.fluxcd.io` to our new website <https://fluxcd.io> work. Changes to this file need to be merged and a manual build triggered in the Netlify App.
