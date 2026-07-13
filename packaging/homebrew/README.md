# Homebrew Packaging

The source repository is [`ongyoo/gokub`](https://github.com/ongyoo/gokub).

1. Create the tap repository `ongyoo/homebrew-tap`.
2. Create a fine-grained GitHub token scoped only to that repository with
   **Contents: Read and write**.
3. Add it to the GOKUB repository as the Actions secret
   `HOMEBREW_TAP_TOKEN`.
4. Push a release tag. The workflow renders `gokub.rb`, attaches it to the GitHub
   release, and commits it to `Formula/gokub.rb` in the tap.
5. Test with `brew install ongyoo/tap/gokub`.

Users can then install with:

```bash
brew install ongyoo/tap/gokub
```

Homebrew validates release archives against the SHA-256 values in the formula.
If the secret is not configured, release publishing still succeeds and the rendered
`gokub.rb` remains available as a release asset for manual tap updates.

To render a formula manually from local GoReleaser output:

```bash
./scripts/render-homebrew-formula.sh v0.2.0 dist/checksums.txt dist/gokub.rb
```
