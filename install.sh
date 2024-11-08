ODA_VERSION=${ODA_VERSION:-v0.0.9}
OS=${OS:-$(uname | tr '[:upper:]' '[:lower:]')}
ARCH=${ARCH:-$(uname -m)}

# Handle architecture translation if not already set
if [[ "$ARCH" == "x86_64" ]]; then
  ARCH="amd64"
elif [[ "$ARCH" == "arm64" || "$ARCH" == "aarch64" ]]; then
  ARCH="arm64"
fi

# Check if wget is available, otherwise fall back to curl
if command -v wget &> /dev/null; then
  downloader="wget -O oda-$OS-$ARCH.tar.gz"
else
  downloader="curl -L -o oda-$OS-$ARCH.tar.gz"
fi

# Download, unzip, and move binary in one go
$downloader https://github.com/devzero-inc/local-developer-analytics/releases/download/$ODA_VERSION/oda-$OS-$ARCH.tar.gz && \
tar -xvf oda-$OS-$ARCH.tar.gz && \
sudo mv oda /usr/local/bin/oda && \
rm oda-$OS-$ARCH.tar.gz
