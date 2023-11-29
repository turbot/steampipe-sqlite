#!/usr/bin/env bash

set -e

main() {
  # ANSI escape code variables
  BOLD=$(tput bold)
  NORMAL=$(tput sgr0)

  if ! command -v gunzip >/dev/null; then
    echo "Error: 'gunzip' is required." 1>&2
    exit 1
  fi

  if [ "$OS" = "Windows_NT" ]; then
    echo "Error: Windows is not supported yet." 1>&2
    exit 1
  else
    case $(uname -sm) in
    "Darwin x86_64") target="darwin_amd64.so.gz" ;;
    "Darwin arm64") target="darwin_arm64.so.gz" ;;
    "Linux x86_64") target="linux_amd64.so.gz" ;;
    "Linux aarch64") target="linux_arm64.so.gz" ;;
    *) echo "Error: '$(uname -sm)' is not supported yet." 1>&2;exit 1 ;;
    esac
  fi

  # Validate the inputs
  validate_inputs "$@"

  # Generate the URI for the FDW
  if [ "$version" = "latest" ]; then
    uri="https://api.github.com/repos/turbotio/steampipe-plugin-${plugin}/releases/latest"
    asset_name="steampipe_sqlite_extension_${plugin}_${target}"
  else
    uri="https://api.github.com/repos/turbotio/steampipe-plugin-${plugin}/releases/tags/${version}"
    asset_name="steampipe_sqlite_extension_${plugin}_${target}"
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
  gunzip $asset_name

  # get the filename without the .gz extension and rename it to remove the OS and architecture
  filename=$(echo $asset_name | sed 's/\.gz$//')
  mv $filename steampipe_sqlite_extension_${plugin}.so

  # move the .so file to the desired location if provided
  if [ "$location" != "$(pwd)" ]; then
    mv steampipe_sqlite_extension_${plugin}.so $location
  fi

  echo ""
  echo "${BOLD}$asset_name${NORMAL} downloaded and extracted successfully at ${BOLD}$location${NORMAL}."

  # Remove the downloaded tar.gz file
  rm -f $asset_name
}

validate_inputs() {
  # Default values
  plugin=""
  version="latest"  # default version
  location=$(pwd)   # default location to current working directory

  # Regex for version (assuming version starts with 'v' followed by semantic versioning, e.g., v1.0.0, v2.1.3)
  version_regex='^v[0-9]+(\.[0-9]+)*$'

  # Check the number of arguments
  if [ $# -eq 0 ]; then
      echo "Usage: $0 <plugin> [version] [location]"
      exit 1
  elif [ $# -eq 1 ]; then
      plugin=$1
  elif [ $# -eq 2 ]; then
      if [[ $2 =~ $version_regex ]]; then
          plugin=$1
          version=$2
      else
          plugin=$1
          location=$2
      fi
  elif [ $# -eq 3 ]; then
      plugin=$1
      version=$2
      location=$3
  fi
}

# Call the main function to run the script
main "$@"
