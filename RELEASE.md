# Release workflow

The followings are release workflow of new version (They should be automated in future).

1. Strip `-dev` from `Version` 
1. Write `CHANGELOG.md` 
1. Git commit and push to master
1. Put git tag (e.g, `git tag -a v0.1.1 -m "v0.1.1"`)
1. Bump up version with `-dev` suffix 
