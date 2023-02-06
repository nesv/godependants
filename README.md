# godependants

List dependant packages.

Sometimes you have a large, sprawling codebase, and you desperately would like
to know exactly which packages import (or depend on) the current package.
Enter `godependants`.

Suppose you have three packages in your Go module, and the dependency graph
looks like this:

```
foo -> bar -> baz
```

where `foo` imports `bar`, and `bar` imports `baz`.
To know all of the packages that directly and indirectly depend on `baz`, you
can run:

```sh
$ godependants github.com/nesv/godependants/baz
github.com/nesv/godependants/bar
github.com/nesv/godependants/foo
```

Note that the output of `godependants` is not deterministic, nor is it sorted.

## Direct dependencies 

If you would like to only show direct dependants, specify the `-direct` flag:

```sh
$ godependants -direct github.com/nesv/godependants/baz
github.com/nesv/godependants/bar
```

## Shhhh...

`godependants` can be a little noisy.
It prints diagnostic information to STDERR.
To prevent it from doing this, pass the `-quiet` flag.

```sh
$ godependants -quiet ...
```

## Continuous integration

Where `godependants` can shine, is in continuous integration and testing
environments!

Suppose you have a very, _very_ large repository, and would only like to run
tests on the packages that would have been affected by a recent change.
Your first step would be to get the list of all packages that have changed:

```sh
pkgs_changed=$(git diff --name-only --diff-filter=ACM |\
  grep -E '\.go$' |\
  xargs dirname |\
  uniq)
```

Next, take that list, and pass it to `godependants`:

```sh
godependants ${pkgs_changed[*]}
```

Then you can take the resulting list, and pass it to `go test`.

For a really gnarly one-liner:

```sh
git diff --name-only --diff-filter=ACM |\
  grep -E '\.go$' |\
  xargs dirname |\
  sort |\
  uniq |\
  xargs godependants |\
  go test
```
