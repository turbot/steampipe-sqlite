#!/usr/bin/env bash

set -e

main() {
  # ANSI escape code variables
  BOLD=$(tput bold)
  NORMAL=$(tput sgr0)

  if ! command -v tar >/dev/null; then
    echo "Error: 'tar' is required." 1>&2
    exit 1
  fi

  if [ "$OS" = "Windows_NT" ]; then
    echo "Error: Windows is not supported yet." 1>&2
    exit 1
  else
    case $(uname -sm) in
    "Darwin x86_64") target="darwin_amd64.tar.gz" ;;
    "Darwin arm64") target="darwin_arm64.tar.gz" ;;
    "Linux x86_64") target="linux_amd64.tar.gz" ;;
    "Linux aarch64") target="linux_arm64.tar.gz" ;;
    *) echo "Error: '$(uname -sm)' is not supported yet." 1>&2;exit 1 ;;
    esac
  fi

  # Validate the inputs
  validate_inputs "$@"

  # Generate the URI for the FDW
  if [ "$version" = "latest" ]; then
    uri="https://api.github.com/repos/turbotio/steampipe-plugin-${plugin}/releases/latest"
    asset_name="steampipe_sqlite_${plugin}.${target}"
  else
    uri="https://api.github.com/repos/turbotio/steampipe-plugin-${plugin}/releases/tags/${version}"
    asset_name="steampipe_sqlite_${plugin}.${target}"
  fi

  # Read the GitHub Personal Access Token
  GITHUB_TOKEN=${GITHUB_TOKEN:-}  # Assuming GITHUB_TOKEN is set as an environment variable

  # Check if the GITHUB_TOKEN is set
  if [ -z "$GITHUB_TOKEN" ]; then
    echo ""
    echo "Error: GITHUB_TOKEN is not set. Please set your GitHub Personal Access Token as an environment variable." 1>&2
    exit 1
  fi
  AUTH="Authorization: token $GITHUB_TOKEN"

  response=$(curl -sH "$AUTH" $uri)
  id=`echo "$response" | jq --arg asset_name "$asset_name" '.assets[] | select(.name == $asset_name) | .id' |  tr -d '"'`
  GH_ASSET="$uri/releases/assets/$id"

  echo ""
  echo "Downloading ${BOLD}${asset_name}${NORMAL}..."
  curl -#SL -H "$AUTH" -H "Accept: application/octet-stream" \
     "https://api.github.com/repos/turbotio/steampipe-plugin-${plugin}/releases/assets/$id" \
     -o "$asset_name" -L --create-dirs

  # Use gunzip to extract it
  tar -xvf $asset_name

  # move the .so file to the desired location if provided
  if [ "$location" != "$(pwd)" ]; then
    mv steampipe_sqlite_${plugin}.so $location
  fi

  echo ""
  echo "${BOLD}$asset_name${NORMAL} downloaded and extracted successfully at ${BOLD}$location${NORMAL}."

  # Remove the downloaded tar.gz file
  rm -f $asset_name
}

validate_inputs() {
  # Check if plugin is provided as an argument
  if [ $# -eq 0 ] || [ -z "$1" ]; then
    read -p "Enter the plugin name: " plugin
  else
    plugin=$1
  fi

  # Check if version is provided as an argument
  if [ $# -lt 2 ] || [ -z "$2" ]; then
    read -p "Enter version (latest): " version
    version=${version:-latest}  # Default to 'latest' if input is empty
  else
    version=$2
  fi

  # Check if location is provided as an argument
  if [ $# -lt 3 ] || [ -z "$3" ]; then
    read -p "Enter location (current directory): " location
    location=${location:-$(pwd)}  # Default to current directory if input is empty
  else
    location=$3
  fi
}

# Call the main function to run the script
main "$@"
