#!/bin/bash
# Skript pro přípravu Flathub submission
# Spusť po vytvoření GitHub release

set -e

REPO="Lokalizace-Net/czmanager-gui"
ASSET="cz-agent-gui-linux-amd64"

echo "========================================"
echo "  Flathub Submission Preparation"
echo "========================================"
echo ""

# Získej info o posledním release
echo "Stahuji info o posledním release..."
RELEASE_INFO=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest")

VERSION=$(echo "$RELEASE_INFO" | jq -r '.tag_name | sub("^v"; "")')
DOWNLOAD_URL=$(echo "$RELEASE_INFO" | jq -r ".assets[] | select(.name == \"${ASSET}\") | .browser_download_url")
SIZE=$(echo "$RELEASE_INFO" | jq -r ".assets[] | select(.name == \"${ASSET}\") | .size")

if [ "$DOWNLOAD_URL" = "null" ] || [ -z "$DOWNLOAD_URL" ]; then
    echo "CHYBA: Asset ${ASSET} nenalezen v release!"
    exit 1
fi

echo "Verze: ${VERSION}"
echo "URL: ${DOWNLOAD_URL}"
echo "Velikost: ${SIZE}"
echo ""

# Stáhni binárku a spočítej SHA256
echo "Stahuji binárku pro výpočet SHA256..."
TMPFILE=$(mktemp)
curl -sL "$DOWNLOAD_URL" -o "$TMPFILE"
SHA256=$(sha256sum "$TMPFILE" | cut -d' ' -f1)
rm "$TMPFILE"

echo "SHA256: ${SHA256}"
echo ""

# Aktualizuj manifest
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MANIFEST="${SCRIPT_DIR}/net.lokalizace.CZManager.yml"

echo "Aktualizuji manifest..."
sed -i "s|sha256: .*|sha256: ${SHA256}|" "$MANIFEST"
sed -i "s|size: .*|size: ${SIZE}|" "$MANIFEST"
sed -i "s|url: https://github.com/.*/download/.*|url: ${DOWNLOAD_URL}|" "$MANIFEST"

echo ""
echo "========================================"
echo "  Manifest aktualizován!"
echo "========================================"
echo ""
echo "Další kroky pro Flathub submission:"
echo ""
echo "1. Forkni https://github.com/flathub/flathub"
echo "2. git clone --branch=new-pr https://github.com/TVUJ-USER/flathub.git"
echo "3. cd flathub && git checkout -b net.lokalizace.CZManager"
echo "4. Zkopíruj tyto soubory do repo:"
echo "   cp ${SCRIPT_DIR}/net.lokalizace.CZManager.yml ."
echo "   cp ${SCRIPT_DIR}/../net.lokalizace.CZManager.desktop ."
echo "   cp ${SCRIPT_DIR}/../net.lokalizace.CZManager.metainfo.xml ."
echo "   cp ${SCRIPT_DIR}/../icon-256.png ."
echo "5. git add . && git commit -m 'Add net.lokalizace.CZManager'"
echo "6. git push origin net.lokalizace.CZManager"
echo "7. Vytvoř PR do flathub/flathub (base: new-pr)"
echo "   Titulek: 'Add net.lokalizace.CZManager'"
echo ""
