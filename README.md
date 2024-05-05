# Helm watch plugin

`watch` watches your chart (and values file) for changes, depending on given filters, prints updated manifests as they are generated.

> Intended for local usage only, since if specified <repo>/<chart> this plugin won't know where to watch, you could manually unarchive and point to the destination for this to work.

I made this mostly for myself. While I don't intend to make many breaking changes, please open issue on this repo with your suggestions.

If you want to contribute, see [here](#improvements) for potential improvement ideas I am planning to make in the coming days.

## Installation

```sh
helm plugin install https://github.com/debdutdeb/helm-watch
```

To update, run

```sh
helm plugin update https://github.com/debdutdeb/helm-watch
```

## Usage

Help

```sh
Usage of /Users/debdut/Library/helm/plugins/helm-watch/watch/helm-watch-darwin-arm64: helm watch --chart <chart> --kinds <kinds> --names <resources> [--release-name [release]] -- [optional args for "helm template" command
  -chart string
        --chart <path to local chart>
  -kinds string
        --kinds <kind of resource to watch> (no shorthands, separated by comma)
  -names string
        --names <name of resource to watch> (regex, separated by comma)
  -release-name string
        --release-name <name of release> or use "-- --generate-name"
```

The main two options are `--kinds` and `--names` together. Pass kinds and resource names separated by commas. For each kind, the name is the one at the respective index.
For example if you pass `--kinds Deployment,service --names nginx,lb`, helm-watch will track the deployment named `nginx` and service named `lb`.

If you omit the names altogether, or you don't have name specified for a kind, plugin will default to `.+` regular expression, i.e. it will track everything of that kind.

Did I say `--names` accepts regular expressions? Yes it does.

So, to track all ingresses, use `helm watch --chart <path> --kinds ingress -- --generate-name`.

Nothing else to it I'm afraid.

## Improvements

1. Supporting short names and plurals for kinds. For example instead of `--kinds service`, being able to `--kind svc`.
2. I wanted to make this `helm template` drop in replacement. You should be able to just switch `template` for `watch` with filter arguments for this to work. I didn't go with this idea since then either I'd have to import the cmd (`cobra.Command`) for `template` which I don't even think is exported, or copy it, or somehow track boolean and non boolean arguments. Pain is, there is no specific flags for release name and chart path. They are positional. Without knowing which ones are boolean flags which ones aren't, I can't skip them and grab just the name and chart. Maybe there is a better way that my stupid self did not think of. But I did not entertain this idea any more than 10 seconds because of the reasons mentioned.
3. Label selectors, for sure I'd love.

Feel free to contribute any other ideas you might have. I don't expect this tool to become **wow-that-big-thing**, it's just a script in Go. Anything that is helpful to some is a win, as long as it's (idea or implementation) not objectively bad. I *might* impose some opinionated review changes on code style, please don't take it as a dig at your skill or anything. Mostly, it's my stubbornness, and the fact that I *do* have some say here.
