# Publishing a New Release

1. Make sure all changes are merged to `main` and working.

2. Create and push a version tag:
   ```bash
   git tag v1.2.0
   git push origin v1.2.0
   ```

3. GitHub Actions automatically:
   - Compiles binaries for Linux and macOS (amd64 + arm64)
   - Packages a `.deb` for Ubuntu
   - Creates a GitHub Release with all assets attached
   - Generates `checksums.txt`

   Watch progress: https://github.com/saimageshvar/pretty-git/actions

4. Once the release is live, users on any Ubuntu machine can install or update with:
   ```bash
   curl -sSL https://raw.githubusercontent.com/saimageshvar/pretty-git/main/install.sh | sudo bash
   ```

## Version naming

Follow [semver](https://semver.org): `vMAJOR.MINOR.PATCH`

- `v1.0.0` → first stable release
- `v1.1.0` → new feature
- `v1.1.1` → bug fix
