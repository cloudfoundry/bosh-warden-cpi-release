## bosh-warden-cpi-release

* Documentation: [bosh.io/docs](https://bosh.io/docs)

This is a BOSH release for the Garden (used to be called Warden) CPI. It's used in bosh-lite.

## Development

Run `./src/github.com/cppforlife/bosh-warden-cpi/bin/test` for unit tests.

### Light stemcell image references

Light stemcells must provide a SHA256 digest in `cloud_properties.digest` or use
an image reference already pinned with `@sha256:...`. When both are present they
must agree. The CPI stores the immutable reference in the stemcell CID and passes
it to Garden for content-addressed pulling. Existing mutable `light://...:tag`
CIDs must be replaced by re-uploading the light stemcell with its digest.
