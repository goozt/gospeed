# Release setup — one-time manual actions

After the gobin-template adaptation, each package-manager publisher in
`.goreleaser.yaml` is **secret-gated**: if its credential is missing at release
time, the publisher silently skips (look for `pipe skipped` in the GoReleaser
log) and the release succeeds. So this list can be tackled in any order, in
multiple sittings — partial completion never breaks a release.

The five sections below are independent. Each lists the goal, the exact secret
or variable name the workflow expects, and the steps. **Names matter** —
secrets must be named exactly as written or the workflow will treat them as
unset.

---

## 1. AUR (Arch User Repository)

**What this enables**: `yay -S gospeed-bin` and `yay -S gospeed-server-bin`
auto-publish on every `git tag`.

**Required secret**: `AUR_SSH_PRIVATE_KEY` (the PEM-encoded private SSH key).

### Steps

1. **AUR account** — register at <https://aur.archlinux.org/register> if you
   don't have one (one-time, free).

2. **Generate a dedicated SSH key** for AUR publishing. Don't reuse your
   personal SSH key.

   ```sh
   ssh-keygen -t ed25519 -f ~/.ssh/aur_gospeed -C "gospeed-aur" -N ""
   ```

   This writes `~/.ssh/aur_gospeed` (private) and `~/.ssh/aur_gospeed.pub`
   (public).

3. **Register the public key** on your AUR account: <https://aur.archlinux.org/account/>
   → *My Account* → paste the contents of `~/.ssh/aur_gospeed.pub` into the
   *SSH Public Key* box → Update.

4. **Submit two empty initial PKGBUILDs**. The AUR requires a package to exist
   before GoReleaser can push updates. For each of `gospeed-bin` and
   `gospeed-server-bin`:

   ```sh
   git clone ssh://aur@aur.archlinux.org/gospeed-bin.git
   cd gospeed-bin
   cat > PKGBUILD <<'EOF'
   # Maintainer: goozt <goozt@users.noreply.github.com>
   pkgname=gospeed-bin
   pkgver=0.0.0
   pkgrel=1
   pkgdesc='Placeholder — replaced by GoReleaser on the next tag.'
   arch=('x86_64')
   license=('MIT')
   url='https://gospeed.goozt.org'
   EOF
   makepkg --printsrcinfo > .SRCINFO
   git add PKGBUILD .SRCINFO
   git commit -m "initial"
   git push
   ```

   Repeat with `gospeed-server-bin` (swap the name everywhere).

   > If the `git clone ssh://aur@…` fails with "Permission denied (publickey)",
   > the key isn't registered yet — re-check step 3.

5. **Add the private key as a GitHub secret**.

   ```sh
   gh secret set AUR_SSH_PRIVATE_KEY --repo goozt/gospeed < ~/.ssh/aur_gospeed
   ```

   (Or via the GitHub UI: repo → Settings → Secrets and variables → Actions →
   *New repository secret* → name `AUR_SSH_PRIVATE_KEY`, value = entire PEM
   contents including the `-----BEGIN OPENSSH PRIVATE KEY-----` / `END` lines.)

**Verify**: tag a test release (`v1.4.0-rc1`). The release log should show
`arch user repositories` writing both packages and pushing to `aur.archlinux.org`.

---

## 2. Nix (NUR — Nix User Repository)

**What this enables**: `nix run github:goozt/nur-packages#gospeed` auto-updates
on every tag.

**Required secret**: `NIX_TAP_GITHUB_TOKEN` (a GitHub PAT with `repo` scope on
the NUR repo).

### Steps

1. **Create the NUR repo** — at <https://github.com/new>, name it
   `nur-packages`, owner `goozt`, public, add a README. Leave it otherwise
   empty.

2. **Create a Personal Access Token** with write access to that repo.

   - Easiest: <https://github.com/settings/tokens/new> → name `gospeed-nix-tap`,
     scope `repo` (classic PAT). Expiration: your preference — set a calendar
     reminder if not "no expiration".
   - Fine-grained alternative: <https://github.com/settings/personal-access-tokens/new>
     → only-select `goozt/nur-packages`, *Repository permissions* → *Contents:
     Read and write*.

3. **Add the token as a GitHub secret**.

   ```sh
   gh secret set NIX_TAP_GITHUB_TOKEN --repo goozt/gospeed
   # paste PAT when prompted
   ```

4. (Optional) **Register the NUR** with the upstream index at
   <https://github.com/nix-community/NUR> so users discover it from `nixpkgs`.
   Submit a PR to NUR's `repos.json` adding an entry for `goozt/nur-packages`.
   Not required for the release flow to work.

**Verify**: after the next tag, `goozt/nur-packages` should gain a commit from
`goreleaserbot` updating `pkgs/gospeed/default.nix`.

> **Local heads-up**: snapshots run on Windows currently skip Nix with
> `nix-hash is not available`. That's harmless — Nix only runs on CI.

---

## 3. Winget (Microsoft package manager)

**What this enables**: `winget install goozt.gospeed` and
`winget install goozt.gospeed-server` auto-PR to `microsoft/winget-pkgs` on
every tag. Microsoft reviews the first PR manually; subsequent updates
auto-merge as long as the manifest schema is unchanged.

**Required secret**: `WINGET_GITHUB_TOKEN` (a PAT with `repo` + `workflow`
scopes — the `workflow` scope is needed because PRs into `winget-pkgs` touch
workflow files).

### Steps

1. **Fork `microsoft/winget-pkgs`** to your account/org:
   <https://github.com/microsoft/winget-pkgs/fork>. Keep the default name
   (`winget-pkgs`) so the GoReleaser config finds it. **Owner must be `goozt`**
   to match `.goreleaser.yaml`.

2. **Create a Personal Access Token** with `repo` + `workflow` scopes:
   <https://github.com/settings/tokens/new>. Name it `gospeed-winget`.

3. **Add the token as a GitHub secret**.

   ```sh
   gh secret set WINGET_GITHUB_TOKEN --repo goozt/gospeed
   ```

4. **Reserve the package identifiers**. The Winget repo enforces uniqueness of
   `Publisher.PackageName`. For a new identifier the first PR is reviewed by a
   human; if a Microsoft reviewer wants you to change the publisher prefix
   (rare for original projects), they'll comment on the PR.

   Identifiers used by this project:
   - `goozt.gospeed`
   - `goozt.gospeed-server`

   You don't need to pre-reserve — just be ready for the first auto-opened PR
   to receive review feedback. If the reviewers reject the identifier, update
   `package_identifier:` in `.goreleaser.yaml` and re-tag.

**Verify**: after the next tag, two PRs should appear at
<https://github.com/microsoft/winget-pkgs/pulls?q=is:pr+author:goozt>.

---

## 4. Fury.io (hosted apt / yum / apk + PyPI)

**What this enables**:
- `sudo apt install gospeed gospeed-server`
- `sudo dnf install gospeed gospeed-server`
- `sudo apk add gospeed gospeed-server`
- `pipx install --index-url https://pypi.fury.io/<account>/ gospeed`

**Required**:
- secret `FURY_PUSH_TOKEN`
- variable `FURY_ACCOUNT` (note: **variable**, not secret — it's in URLs)

### Steps

1. **Sign up** at <https://gemfury.com/signup>. Pick an account name — this
   becomes part of the install URL (`https://apt.fury.io/<account>/`). Public
   repos are free with no per-package limit.

2. **Create a push token** at <https://manage.fury.io/manage/tokens/push>.
   Treat it like a password — anyone with it can write to your Fury account.

3. **Add the GitHub variable** (Settings → Secrets and variables → Actions →
   **Variables** tab):

   ```sh
   gh variable set FURY_ACCOUNT --repo goozt/gospeed --body "your-fury-account-name"
   ```

4. **Add the GitHub secret** (Secrets tab):

   ```sh
   gh secret set FURY_PUSH_TOKEN --repo goozt/gospeed
   ```

5. (Optional) **Update README links**. The README currently uses `goozt` as the
   Fury account placeholder. If your chosen Fury account name is different,
   search/replace `apt.fury.io/goozt`, `yum.fury.io/goozt`, `alpine.fury.io/goozt`,
   and `pypi.fury.io/goozt` in `README.md`.

6. (Production hardening, optional) **Enable GPG signing** for apt/yum
   repos so users don't need `trusted=yes` / `gpgcheck=0`. Follow
   <https://gemfury.com/help/sign-packages/>.

**Verify**: after the next tag, the `Release (Fury.io)` workflow should show
both jobs (`fury-linux-packages`, `fury-pypi`) green. The Fury dashboard at
<https://manage.fury.io/manage/packages> should list the new packages and the
wheel.

> **Rotation**: if the push token leaks, revoke at the same URL and update the
> `FURY_PUSH_TOKEN` secret. The token is the only credential — rotating locks
> out anyone who saw the old one.

---

## 5. npm trusted publishing

**What this enables**: `npm install -g @goozt/gospeed` and
`npm install -g @goozt/gospeed-server` auto-publish via OIDC (no NPM_TOKEN
secret needed — npmjs.com trusts GitHub's identity token).

The `@goozt/gospeed` package is already configured (the old release flow used
it). The new `@goozt/gospeed-server` needs to be added.

### Steps

1. **Sign in** to <https://www.npmjs.com> as the npm user with publish rights
   to the `@goozt` scope.

2. For **each** package (`@goozt/gospeed` and `@goozt/gospeed-server`):

   - If the package doesn't exist yet on npm, you'll need a first manual
     publish (the trusted-publishing config can't be set on a non-existent
     package). For `gospeed-server` this is likely the case.

     ```sh
     cd npm/gospeed-server
     # bump version to a placeholder you can overwrite, e.g. 0.0.1
     npm version 0.0.1 --no-git-tag-version
     npm publish --access public
     ```

     Then `npm unpublish @goozt/gospeed-server@0.0.1 --force` within 72h if you
     want a clean slate (npmjs allows unpublishing recently-created versions).

   - Open the package settings page on npmjs.com:
     <https://www.npmjs.com/package/@goozt/gospeed-server/access>.

   - Scroll to *Publishing access* → *Trusted Publisher* → *Add trusted
     publisher*.

   - Fill in:
     - **Organization or user**: `goozt`
     - **Repository**: `gospeed`
     - **Workflow filename**: `release-npm.yml`
     - **Environment**: `npm`

   - Save.

3. **Confirm `@goozt/gospeed` is configured the same way**. The new workflow
   filename is `release-npm.yml` (not `release.yml` as before, where npm was a
   sub-job). If npmjs is still configured to trust `release.yml`, update it to
   `release-npm.yml`.

4. **Confirm the GitHub `npm` environment exists** at
   <https://github.com/goozt/gospeed/settings/environments>. It should already
   exist from the previous setup. No deployment-protection rules need to
   change.

**Verify**: after the next tag, the `Release (npm)` workflow runs and both
packages publish without an `NPM_TOKEN` in the env block. The published
versions should show a provenance attestation on their npmjs.com pages.

---

## Pre-flight checklist (cut a `-rc` tag to test)

Once you've done as many of the above as you want, do a smoke-test release:

```sh
git tag v1.4.0-rc1
git push origin v1.4.0-rc1
```

Then watch <https://github.com/goozt/gospeed/actions> and inspect each
workflow:

| Workflow | What to look for |
|---|---|
| `Release` | All GoReleaser pipes succeed or show `pipe skipped`. Anything that should fire but didn't = missing/typo'd secret. |
| `Release (npm)` | Both `@goozt/gospeed` and `@goozt/gospeed-server` published with `--provenance`. |
| `Release (Fury.io)` | `fury-linux-packages` pushes 16 each of deb/rpm/apk; `fury-pypi` uploads one wheel + one sdist. |

Where to confirm artifacts landed:

- GitHub Releases: <https://github.com/goozt/gospeed/releases/tag/v1.4.0-rc1>
- Homebrew tap: <https://github.com/goozt/homebrew-org/tree/main/Formula>
- Scoop bucket: <https://github.com/goozt/scoop-bucket>
- AUR: <https://aur.archlinux.org/packages/gospeed-bin> + `gospeed-server-bin`
- NUR: <https://github.com/goozt/nur-packages>
- Winget PRs: <https://github.com/microsoft/winget-pkgs/pulls?q=author:goozt>
- Fury dashboard: <https://manage.fury.io/manage/packages>
- npm: <https://www.npmjs.com/package/@goozt/gospeed> + `/@goozt/gospeed-server`

After confirming, delete the `-rc` release and tag, then cut the real version.

---

## Quick reference: GitHub secrets/variables this repo now expects

| Name | Type | Used by | Notes |
|---|---|---|---|
| `GITHUB_TOKEN` | (auto) | all workflows | Auto-injected by Actions, no setup. |
| `PACKAGE_PUBLISHING_TOKEN` | secret | Release (Homebrew, Scoop) | Already configured. PAT with `repo` on `goozt/homebrew-org` and `goozt/scoop-bucket`. |
| `AUR_SSH_PRIVATE_KEY` | secret | Release (AUR) | §1 above. |
| `NIX_TAP_GITHUB_TOKEN` | secret | Release (Nix) | §2 above. PAT with `repo` on `goozt/nur-packages`. |
| `WINGET_GITHUB_TOKEN` | secret | Release (Winget) | §3 above. PAT with `repo` + `workflow`. |
| `FURY_ACCOUNT` | **variable** | Release (Fury.io) | §4 above. Account name only. |
| `FURY_PUSH_TOKEN` | secret | Release (Fury.io) | §4 above. |
| `CODECOV_TOKEN` | secret | CI | Already configured. |

Missing entries cause a silent skip — never a failed release.
