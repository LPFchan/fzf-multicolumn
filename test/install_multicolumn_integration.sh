#!/bin/sh
set -eu
binary=${1:?built binary required}
case "$binary" in /*) ;; *) binary="$(pwd)/$binary" ;; esac
base=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/fzf-installer.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
mkdir -p "$work/repo/bin" "$work/mocks" "$work/archive"
cp "$base/install" "$work/repo/install"
cp "$binary" "$work/repo/bin/fzf"
cp "$binary" "$work/archive/fzf-multicolumn"
tar -czf "$work/fork.tgz" -C "$work/archive" fzf-multicolumn
cat > "$work/mocks/curl" <<'MOCK'
#!/bin/sh
for installer_url do :; done
case "$installer_url" in
  https://github.com/LPFchan/fzf-multicolumn/releases/download/v*/fzf-multicolumn-*.tgz)
    echo "$installer_url" > "$FZF_INSTALL_TEST_REQUEST"
    cat "$FZF_INSTALL_TEST_ARCHIVE" ;;
  *) echo "unexpected download: $installer_url" >&2; exit 1 ;;
esac
MOCK
cat > "$work/mocks/fzf" <<'MOCK'
#!/bin/sh
echo 'stock-fzf-without-grid'
MOCK
cat > "$work/mocks/wget" <<'MOCK'
#!/bin/sh
exit 1
MOCK
chmod +x "$work/mocks/curl" "$work/mocks/fzf" "$work/mocks/wget"
export FZF_INSTALL_TEST_ARCHIVE="$work/fork.tgz"
export FZF_INSTALL_TEST_REQUEST="$work/request"
export PATH="$work/mocks:$PATH"
# Configuration setup must not replace a binary built from this checkout.
bash "$work/repo/install" --bin
cmp "$binary" "$work/repo/bin/fzf"
test ! -e "$work/request"
# A stock binary must be replaced by a fork asset, never by stock upstream.
cp "$work/mocks/fzf" "$work/repo/bin/fzf"
bash "$work/repo/install" --bin
cmp "$binary" "$work/repo/bin/fzf"
test -s "$work/request"
