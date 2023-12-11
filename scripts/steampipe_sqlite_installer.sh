#!/bin/sh

set -e

main() {
  # ANSI escape code variables
  BOLD=$(tput bold)
  NORMAL=$(tput sgr0)

  if ! command -v tar >/dev/null 2>&1; then
    echo "Error: 'tar' is required." 1>&2
    exit 1
  fi

  OS=$(uname -s)
  if [ "$OS" = "Windows_NT" ]; then
    echo "Error: Windows is not supported yet." 1>&2
    exit 1
  else
    UNAME_SM=$(uname -sm)
    case "$UNAME_SM" in
    "Darwin x86_64") target="darwin_amd64.tar.gz" ;;
    "Darwin arm64") target="darwin_arm64.tar.gz" ;;
    "Linux x86_64") target="linux_amd64.tar.gz" ;;
    "Linux aarch64") target="linux_arm64.tar.gz" ;;
    *) echo "Error: '$UNAME_SM' is not supported yet." 1>&2; exit 1 ;;
    esac
  fi

  # Validate the inputs
  validate_inputs "$@"

  # Generate the URI for the extension
  asset_name="steampipe_sqlite_${plugin}.${target}"
  if [ "$version" = "latest" ]; then
    uri="https://github.com/turbot/steampipe-plugin-${plugin}/releases/latest/download/${asset_name}"
  else
    uri="https://github.com/turbot/steampipe-plugin-${plugin}/releases/download/${version}/${asset_name}"
  fi

  echo ""
  echo "Downloading ${BOLD}${asset_name}${NORMAL}..."
  if ! curl --fail --progress-bar "$uri"; then
    echo "Could not find version $version"
    exit 1
  fi

  # Use tar to extract it
  tar -xvf $asset_name

  # move the .so file to the desired location if provided
  if [ "$location" != "$(pwd)" ]; then
    mv "steampipe_sqlite_${plugin}.so" "$location"
  fi

  echo ""
  echo "${BOLD}$asset_name${NORMAL} downloaded and extracted successfully at ${BOLD}$location${NORMAL}."

  # Remove the downloaded tar.gz file
  rm -f $asset_name
}

validate_inputs() {
  # Check if plugin is provided as an argument
  if [ $# -eq 0 ] || [ -z "$1" ]; then
    printf "Enter the plugin name: "
    read plugin
  else
    plugin=$1
  fi

  # Check if version is provided as an argument
  if [ $# -lt 2 ] || [ -z "$2" ]; then
    printf "Enter version (latest): "
    read version
    version=${version:-latest}
  else
    version=$2
  fi

  # Check if location is provided as an argument
  if [ $# -lt 3 ] || [ -z "$3" ]; then
    printf "Enter location (current directory): "
    read location
    location=${location:-$(pwd)}
  else
    location=$3
  fi
}

# Call the main function to run the script
main "$@"