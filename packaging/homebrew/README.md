# Homebrew Packaging

The final repository URL has not been selected yet. After it is available:

1. Create a tap repository such as `<owner>/homebrew-tap`.
2. Copy `Formula/gokub.rb.tmpl` to that repository as `Formula/gokub.rb`.
3. Replace `OWNER`, `REPOSITORY`, `VERSION`, and the four SHA-256 placeholders
   with values from the GitHub release and `checksums.txt`.
4. Test with `brew install --build-from-source <owner>/tap/gokub`.

Users can then install with:

```bash
brew install <owner>/tap/gokub
```

Homebrew validates release archives against the SHA-256 values in the formula.
